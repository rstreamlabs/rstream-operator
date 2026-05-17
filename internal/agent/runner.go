// See LICENSE file in the project root for license information.

package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/rstreamlabs/rstream-go"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

type Runner struct {
	Config   agentconfig.Config
	Reporter Reporter
	Health   *HealthState
	Logger   *slog.Logger
}

func (r *Runner) Run(ctx context.Context) error {
	if err := ValidateConfig(r.Config); err != nil {
		return err
	}
	reporter := r.Reporter
	if reporter == nil {
		reporter = NoopReporter{}
	}
	health := r.Health
	if health == nil {
		health = &HealthState{}
	}
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	initial, maximum, err := RetryDurations(r.Config)
	if err != nil {
		return err
	}
	backoff := newBackoff(initial, maximum)
	for {
		if err := ctx.Err(); err != nil {
			health.SetReady(false)
			return nil
		}
		health.SetReady(false)
		_ = reporter.Connecting(ctx)
		err := r.runOnce(ctx, reporter, health, logger)
		if err == nil || errors.Is(err, context.Canceled) {
			health.SetReady(false)
			return nil
		}
		health.SetReady(false)
		_ = reporter.Disconnected(ctx, err)
		retryIn := backoff.Next()
		logger.Warn("Tunnel disconnected; retrying", "error", err, "retry_in", retryIn.String())
		select {
		case <-time.After(retryIn):
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Runner) runOnce(ctx context.Context, reporter Reporter, health *HealthState, logger *slog.Logger) error {
	props, err := TunnelProperties(r.Config.Tunnel)
	if err != nil {
		return err
	}
	client, err := NewRstreamClient(r.Config)
	if err != nil {
		return err
	}
	ctrl, err := client.Connect(ctx, nil)
	if err != nil {
		return fmt.Errorf("connect control channel: %w", err)
	}
	defer ctrl.Close()
	tunnel, err := ctrl.CreateTunnel(ctx, props)
	if err != nil {
		return fmt.Errorf("create tunnel: %w", err)
	}
	defer tunnel.Close()
	resolvedProps, err := tunnel.Properties()
	if err != nil {
		return fmt.Errorf("read tunnel properties: %w", err)
	}
	forwardingAddress, err := tunnel.ForwardingAddress()
	if err != nil {
		return fmt.Errorf("read forwarding address: %w", err)
	}
	online := OnlineStatus{
		TunnelID:          stringPtrValue(resolvedProps.ID),
		Hostname:          firstNonEmpty(stringPtrValue(resolvedProps.Hostname), stringPtrValue(resolvedProps.Host)),
		ForwardingAddress: forwardingAddress,
		RstreamName:       stringPtrValue(resolvedProps.Name),
		Target:            targetAddress(r.Config.Target),
	}
	health.SetReady(true)
	_ = reporter.Online(ctx, online)
	logger.Info("Tunnel online",
		"tunnel_id", online.TunnelID,
		"forwarding_address", online.ForwardingAddress,
		"target", online.Target,
	)
	if l, ok := tunnel.(net.Listener); ok {
		return serveWithContext(ctx, l.Close, func() error {
			return serveTCP(l, r.Config.Target, logger)
		})
	}
	if pl, ok := tunnel.(rstream.PacketListener); ok {
		return serveWithContext(ctx, pl.Close, func() error {
			return serveUDP(pl, r.Config.Target, logger)
		})
	}
	return fmt.Errorf("tunnel does not implement net.Listener or rstream.PacketListener")
}

func serveWithContext(ctx context.Context, closeFn func() error, fn func() error) error {
	errCh := make(chan error, 1)
	go func() { errCh <- fn() }()
	select {
	case <-ctx.Done():
		_ = closeFn()
		<-errCh
		return context.Canceled
	case err := <-errCh:
		return err
	}
}

func serveTCP(listener net.Listener, target agentconfig.TargetConfig, logger *slog.Logger) error {
	for {
		inbound, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go proxyTCP(inbound, target, logger)
	}
}

func proxyTCP(inbound net.Conn, target agentconfig.TargetConfig, logger *slog.Logger) {
	defer inbound.Close()
	dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	outbound, err := dialer.Dial("tcp", targetAddress(target))
	if err != nil {
		logger.Debug("Failed to dial target", "error", err, "target", targetAddress(target))
		return
	}
	defer outbound.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(outbound, inbound); done <- struct{}{} }()
	go func() { _, _ = io.Copy(inbound, outbound); done <- struct{}{} }()
	<-done
}

func serveUDP(listener rstream.PacketListener, target agentconfig.TargetConfig, logger *slog.Logger) error {
	for {
		inbound, remote, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go proxyUDP(inbound, remote, target, logger)
	}
}

func proxyUDP(inbound net.PacketConn, remote net.Addr, target agentconfig.TargetConfig, logger *slog.Logger) {
	defer inbound.Close()
	udpAddr, err := net.ResolveUDPAddr("udp", targetAddress(target))
	if err != nil {
		logger.Debug("Failed to resolve UDP target", "error", err, "target", targetAddress(target))
		return
	}
	outbound, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logger.Debug("Failed to dial UDP target", "error", err, "target", targetAddress(target))
		return
	}
	defer outbound.Close()
	done := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := inbound.ReadFrom(buf)
			if err != nil {
				break
			}
			if _, err := outbound.Write(buf[:n]); err != nil {
				break
			}
		}
		done <- struct{}{}
	}()
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := outbound.Read(buf)
			if err != nil {
				break
			}
			if _, err := inbound.WriteTo(buf[:n], remote); err != nil {
				break
			}
		}
		done <- struct{}{}
	}()
	<-done
}

func targetAddress(target agentconfig.TargetConfig) string {
	return net.JoinHostPort(strings.TrimSpace(target.Host), strings.TrimSpace(target.Port))
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

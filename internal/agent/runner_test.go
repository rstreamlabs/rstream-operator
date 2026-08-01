// See LICENSE file in the project root for license information.

package agent

import (
	"errors"
	"log/slog"
	"net"
	"testing"

	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

var errAcceptStopped = errors.New("accept stopped")

type timeoutError struct{}

func (timeoutError) Error() string {
	return "timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return false
}

type timeoutTCPListener struct {
	calls int
}

func (l *timeoutTCPListener) Accept() (net.Conn, error) {
	l.calls++
	if l.calls == 1 {
		return nil, timeoutError{}
	}
	return nil, errAcceptStopped
}

func (*timeoutTCPListener) Close() error {
	return nil
}

func (*timeoutTCPListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

type timeoutPacketListener struct {
	calls int
}

func (l *timeoutPacketListener) Accept() (net.PacketConn, net.Addr, error) {
	l.calls++
	if l.calls == 1 {
		return nil, nil, timeoutError{}
	}
	return nil, nil, errAcceptStopped
}

func (*timeoutPacketListener) Close() error {
	return nil
}

func (*timeoutPacketListener) Addr() net.Addr {
	return &net.UDPAddr{}
}

func TestServeTCPRetriesTimeoutAndReturnsTerminalError(t *testing.T) {
	listener := &timeoutTCPListener{}
	err := serveTCP(listener, agentconfig.TargetConfig{}, slog.Default())
	if !errors.Is(err, errAcceptStopped) {
		t.Fatalf("serveTCP() error = %v, want %v", err, errAcceptStopped)
	}
	if listener.calls != 2 {
		t.Fatalf("Accept() calls = %d, want 2", listener.calls)
	}
}

func TestServeUDPRetriesTimeoutAndReturnsTerminalError(t *testing.T) {
	listener := &timeoutPacketListener{}
	err := serveUDP(listener, agentconfig.TargetConfig{}, slog.Default())
	if !errors.Is(err, errAcceptStopped) {
		t.Fatalf("serveUDP() error = %v, want %v", err, errAcceptStopped)
	}
	if listener.calls != 2 {
		t.Fatalf("Accept() calls = %d, want 2", listener.calls)
	}
}

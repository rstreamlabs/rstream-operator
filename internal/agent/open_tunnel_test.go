// See LICENSE file in the project root for license information.

package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rstreamlabs/rstream-go"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

type recordingControl struct {
	rstream.ControlChannel
	requests []rstream.TunnelProperties
	current  *recordingTunnel
	domain   string
	closeErr error
}

func (c *recordingControl) CreateTunnel(_ context.Context, props rstream.TunnelProperties) (rstream.Tunnel, error) {
	c.requests = append(c.requests, props)
	if c.current != nil && !c.current.closed {
		return nil, errors.New("single tunnel quota exhausted")
	}
	if props.Hostname == nil {
		props.Hostname = &c.domain
	}
	c.current = &recordingTunnel{props: props, closeErr: c.closeErr}
	return c.current, nil
}

type recordingTunnel struct {
	rstream.Tunnel
	props    rstream.TunnelProperties
	closed   bool
	closeErr error
}

func (t *recordingTunnel) Properties() (rstream.TunnelProperties, error) { return t.props, nil }
func (t *recordingTunnel) Close() error {
	t.closed = true
	return t.closeErr
}

func TestOpenTunnelUsesAuthoritativeServingDomain(t *testing.T) {
	const regional = "project.region.example:8443"
	const global = "project.global.example:443"
	const domain = "deadbeef.t.global.example"
	custom := "custom.example"
	private := false
	tcp := rstream.ProtocolTCP
	for _, tc := range []struct {
		name       string
		connection agentconfig.ConnectionConfig
		props      rstream.TunnelProperties
		requests   int
		wantHost   string
	}{
		{"managed project", agentconfig.ConnectionConfig{Engine: regional, StableDomainEngine: global}, rstream.TunnelProperties{}, 1, "-project.t.global.example"},
		{"direct regional address", agentconfig.ConnectionConfig{Engine: regional}, rstream.TunnelProperties{}, 2, "-project.t.global.example"},
		{"reconnect or explicit name", agentconfig.ConnectionConfig{Engine: regional}, rstream.TunnelProperties{Hostname: &custom}, 1, custom},
		{"private", agentconfig.ConnectionConfig{Engine: regional}, rstream.TunnelProperties{Publish: &private}, 1, ""},
		{"tcp", agentconfig.ConnectionConfig{Engine: regional}, rstream.TunnelProperties{Protocol: &tcp}, 1, ""},
		{"literal address", agentconfig.ConnectionConfig{Engine: "[::1]:443"}, rstream.TunnelProperties{}, 1, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := &recordingControl{domain: domain}
			tunnel, err := openTunnel(context.Background(), ctrl, tc.props, tc.connection)
			if err != nil || tunnel != ctrl.current || ctrl.current.closed || len(ctrl.requests) != tc.requests {
				t.Fatalf("openTunnel returned %v, requests %d, error %v", tunnel, len(ctrl.requests), err)
			}
			if tc.requests == 2 && ctrl.requests[0].Hostname != nil {
				t.Fatal("direct engine discovery requested a hostname")
			}
			host := stringPtrValue(ctrl.requests[tc.requests-1].Hostname)
			if (tc.wantHost == "" && host != "") || !strings.HasSuffix(host, tc.wantHost) {
				t.Errorf("requested hostname %q, want suffix %q", host, tc.wantHost)
			}
		})
	}
}

func TestOpenTunnelRequiresSuccessfulDiscoveryAndClose(t *testing.T) {
	for _, ctrl := range []*recordingControl{{domain: "unrelated.example"}, {domain: "a.t.global.example", closeErr: errors.New("close failed")}} {
		_, err := openTunnel(context.Background(), ctrl, rstream.TunnelProperties{}, agentconfig.ConnectionConfig{Engine: "project.region.example:443"})
		if err == nil || len(ctrl.requests) != 1 || !ctrl.current.closed {
			t.Fatalf("error %v, requests %d, closed %t", err, len(ctrl.requests), ctrl.current.closed)
		}
	}
}

// See LICENSE file in the project root for license information.

package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rstreamlabs/rstream-go"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
)

func TestValidateConfigRequiresAuthContext(t *testing.T) {
	err := ValidateConfig(agentconfig.Config{})
	if err == nil || !strings.Contains(err.Error(), "connection.engine") {
		t.Fatalf("ValidateConfig() error = %v, want engine error", err)
	}
}

func TestTunnelPropertiesMapsHTTPAuth(t *testing.T) {
	token := true
	rstreamAuth := false
	props, err := TunnelProperties(agentconfig.TunnelConfig{
		Name:     "demo",
		Publish:  &token,
		Protocol: "http",
		HTTP: &agentconfig.HTTPConfig{
			Version: "http/1.1",
			Auth: &agentconfig.HTTPAuthConfig{
				Token:   &token,
				Rstream: &rstreamAuth,
			},
		},
	})
	if err != nil {
		t.Fatalf("TunnelProperties() error = %v", err)
	}
	if props.Name == nil || *props.Name != "demo" {
		t.Fatalf("Name = %#v, want demo", props.Name)
	}
	if props.TokenAuth == nil || !*props.TokenAuth {
		t.Fatalf("TokenAuth = %#v, want true", props.TokenAuth)
	}
	if props.RstreamAuth == nil || *props.RstreamAuth {
		t.Fatalf("RstreamAuth = %#v, want false", props.RstreamAuth)
	}
}

func TestTunnelPropertiesMapsDatagramGuaranteedDelivery(t *testing.T) {
	guaranteedDelivery := true
	props, err := TunnelProperties(agentconfig.TunnelConfig{
		Name:                       "udp",
		Type:                       "datagram",
		Protocol:                   "quic",
		DatagramGuaranteedDelivery: &guaranteedDelivery,
	})
	if err != nil {
		t.Fatalf("TunnelProperties() error = %v", err)
	}
	if props.DatagramGuaranteedDelivery == nil || !*props.DatagramGuaranteedDelivery {
		t.Fatalf("DatagramGuaranteedDelivery = %#v, want true", props.DatagramGuaranteedDelivery)
	}
}

func TestTunnelPropertiesMapsCrossRegionRoutingForHTTP(t *testing.T) {
	allow := true
	props, err := TunnelProperties(agentconfig.TunnelConfig{Name: "web", Protocol: "http", AllowCrossRegionRouting: &allow})
	if err != nil {
		t.Fatalf("TunnelProperties() error = %v", err)
	}
	if props.AllowCrossRegionRouting == nil || !*props.AllowCrossRegionRouting {
		t.Fatalf("AllowCrossRegionRouting = %#v, want true", props.AllowCrossRegionRouting)
	}
}

func TestTunnelPropertiesRejectsDatagramGuaranteedDeliveryForBytestream(t *testing.T) {
	guaranteedDelivery := true
	_, err := TunnelProperties(agentconfig.TunnelConfig{
		Name:                       "web",
		Type:                       "bytestream",
		DatagramGuaranteedDelivery: &guaranteedDelivery,
	})
	if err == nil || !strings.Contains(err.Error(), "requires tunnel type") {
		t.Fatalf("TunnelProperties() error = %v, want type error", err)
	}
}

func TestTunnelPropertiesMapsPublishedTCP(t *testing.T) {
	port := uint32(10042)
	props, err := TunnelProperties(agentconfig.TunnelConfig{Name: "ssh", Protocol: "tcp", TCPPort: &port})
	if err != nil {
		t.Fatalf("TunnelProperties() error = %v", err)
	}
	if props.Protocol == nil || *props.Protocol != rstream.ProtocolTCP {
		t.Fatalf("Protocol = %#v, want tcp", props.Protocol)
	}
	if props.Port == nil || *props.Port != port {
		t.Fatalf("Port = %#v, want %d", props.Port, port)
	}
	if props.Publish == nil || !*props.Publish {
		t.Fatalf("Publish = %#v, want true", props.Publish)
	}
	if props.Type == nil || *props.Type != rstream.TunnelTypeBytestream {
		t.Fatalf("Type = %#v, want bytestream", props.Type)
	}
}

func TestTunnelPropertiesRejectsInvalidPublishedTCPSettings(t *testing.T) {
	port := uint32(10042)
	unpublished := false
	for _, test := range []struct {
		name string
		cfg  agentconfig.TunnelConfig
		want string
	}{
		{name: "port without tcp", cfg: agentconfig.TunnelConfig{Name: "ssh", Protocol: "tls", TCPPort: &port}, want: "tcpPort requires"},
		{name: "unpublished", cfg: agentconfig.TunnelConfig{Name: "ssh", Protocol: "tcp", Publish: &unpublished}, want: "requires a published"},
		{name: "hostname", cfg: agentconfig.TunnelConfig{Name: "ssh", Protocol: "tcp", Hostname: "ssh.example.com"}, want: "does not support"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := TunnelProperties(test.cfg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("TunnelProperties() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestTransportModes(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *agentconfig.TransportConfig
		want any
	}{
		{name: "default", cfg: &agentconfig.TransportConfig{}, want: &rstream.AutoTransport{}},
		{name: "auto", cfg: &agentconfig.TransportConfig{Mode: "auto"}, want: &rstream.AutoTransport{}},
		{name: "tls", cfg: &agentconfig.TransportConfig{Mode: "tls"}, want: &rstream.Transport{}},
		{name: "quic", cfg: &agentconfig.TransportConfig{Mode: "quic"}, want: &rstream.QUICTransport{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := transport(test.cfg)
			if err != nil {
				t.Fatalf("transport() error = %v", err)
			}
			if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", test.want) {
				t.Fatalf("transport() = %T, want %T", got, test.want)
			}
		})
	}
}

func TestTransportRejectsInvalidMode(t *testing.T) {
	_, err := transport(&agentconfig.TransportConfig{Mode: "sctp"})
	if err == nil || !strings.Contains(err.Error(), "invalid tunnel transport") {
		t.Fatalf("transport() error = %v, want invalid tunnel transport", err)
	}
}

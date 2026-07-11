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

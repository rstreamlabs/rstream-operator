// See LICENSE file in the project root for license information.

package resources

import (
	"strings"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildAgentConfigInfersHTTP3Datagram(t *testing.T) {
	tunnel := &tunnelsv1alpha1.RstreamTunnel{}
	tunnel.Name = "web"
	tunnel.Namespace = "demo"
	tunnel.UID = types.UID("uid-1")
	tunnel.Spec = tunnelsv1alpha1.RstreamTunnelSpec{
		Target: tunnelsv1alpha1.RstreamTunnelTarget{
			Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromString("http")},
		},
		Protocol: tunnelsv1alpha1.ProtocolHTTP,
		HTTP: &tunnelsv1alpha1.HTTPSpec{
			Version: tunnelsv1alpha1.HTTPVersion3,
		},
		Labels: map[string]string{"app": "web"},
	}
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "project.c.example.com:443"},
	}
	target := ResolvedTarget{Host: "web.demo.svc", Port: 8080, Protocol: corev1.ProtocolUDP}
	cfg := BuildAgentConfig(tunnel, connection, target, "web.example.com", "project.c.example.com:443")
	if cfg.Tunnel.Type != "datagram" {
		t.Fatalf("Tunnel.Type = %q, want datagram", cfg.Tunnel.Type)
	}
	if cfg.Tunnel.HTTP == nil || cfg.Tunnel.HTTP.Version != "h3" {
		t.Fatalf("Tunnel.HTTP.Version = %#v, want h3", cfg.Tunnel.HTTP)
	}
	if got := cfg.Tunnel.Labels["rstream.kubernetes.uid"]; got != "uid-1" {
		t.Fatalf("uid label = %q, want uid-1", got)
	}
	if got := cfg.Target.Port; got != "8080" {
		t.Fatalf("Target.Port = %q, want 8080", got)
	}
}

func TestMarshalAgentConfigDoesNotContainTokenSecretValue(t *testing.T) {
	cfg := BuildAgentConfig(
		&tunnelsv1alpha1.RstreamTunnel{
			Spec: tunnelsv1alpha1.RstreamTunnelSpec{Protocol: tunnelsv1alpha1.ProtocolHTTP},
		},
		&tunnelsv1alpha1.RstreamConnection{Spec: tunnelsv1alpha1.RstreamConnectionSpec{Engine: "engine.example.com:443"}},
		ResolvedTarget{Host: "svc.ns.svc", Port: 8080, Protocol: corev1.ProtocolTCP},
		"host.example.com",
		"engine.example.com:443",
	)
	cfg.Tunnel.Name = "ns-web"
	out, err := MarshalAgentConfig(cfg)
	if err != nil {
		t.Fatalf("MarshalAgentConfig() error = %v", err)
	}
	if !strings.Contains(out, "connection:") || strings.Contains(out, "Connection:") {
		t.Fatalf("agent config did not use YAML field names:\n%s", out)
	}
	if strings.Contains(out, "token") {
		t.Fatalf("agent config unexpectedly contains token-like text:\n%s", out)
	}
}

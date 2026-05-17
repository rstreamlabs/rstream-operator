// See LICENSE file in the project root for license information.

package agent

import (
	"strings"
	"testing"

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

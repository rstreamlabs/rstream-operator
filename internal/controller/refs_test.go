// See LICENSE file in the project root for license information.

package controller

import (
	"reflect"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
)

func TestSecretRefsSeparatesManagerAndAgentSecrets(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{Name: "credentials", Key: "token"},
			ControlPlaneHeaders: []tunnelsv1alpha1.ControlPlaneHeader{
				{Name: "x-deployment-bypass", ValueSecretRef: tunnelsv1alpha1.SecretKeyRef{Name: "control-plane", Key: "bypass"}},
				{Name: "x-shared-secret", ValueSecretRef: tunnelsv1alpha1.SecretKeyRef{Name: "credentials", Key: "token"}},
			},
		},
	}
	if got, want := secretRefs(connection), []secretRef{{Name: "control-plane", Key: "bypass"}, {Name: "credentials", Key: "token"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("secretRefs() = %#v, want %#v", got, want)
	}
	if got, want := agentSecretRefs(connection), []secretRef{{Name: "credentials", Key: "token"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("agentSecretRefs() = %#v, want %#v", got, want)
	}
}

func TestAddControlPlaneHeadersMapsSharedSecretValue(t *testing.T) {
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			ControlPlaneHeaders: []tunnelsv1alpha1.ControlPlaneHeader{
				{Name: "x-first", ValueSecretRef: tunnelsv1alpha1.SecretKeyRef{Name: "control-plane", Key: "bypass"}},
				{Name: "x-second", ValueSecretRef: tunnelsv1alpha1.SecretKeyRef{Name: "control-plane", Key: "bypass"}},
			},
		},
	}
	headers := make(map[string]string)
	addControlPlaneHeaders(headers, connection, secretRef{Name: "control-plane", Key: "bypass"}, "  secret  ")
	if got, want := headers, map[string]string{"x-first": "secret", "x-second": "secret"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("headers = %#v, want %#v", got, want)
	}
}

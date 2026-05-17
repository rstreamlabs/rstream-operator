// See LICENSE file in the project root for license information.

package resources

import (
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestApplyDeploymentUsesRestrictedSecurityAndSecretRefs(t *testing.T) {
	tunnel := &tunnelsv1alpha1.RstreamTunnel{}
	tunnel.Name = "web"
	tunnel.Namespace = "demo"
	tunnel.UID = types.UID("uid-1")
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			Engine: "engine.example.com:443",
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{
				Name: "rstream-token",
				Key:  "token",
			},
		},
	}
	var dep appsv1.Deployment
	err := ApplyDeployment(&dep, tunnel, connection, BuildOptions{
		AgentImage:             "example.com/rstream-operator:test",
		ConfigYAML:             "connection: {}\n",
		SecretResourceVersions: []string{"rstream-token:1"},
	})
	if err != nil {
		t.Fatalf("ApplyDeployment() error = %v", err)
	}
	if dep.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Fatalf("strategy = %s, want Recreate", dep.Spec.Strategy.Type)
	}
	container := dep.Spec.Template.Spec.Containers[0]
	if len(container.Command) != 1 || container.Command[0] != "/rstream-agent" {
		t.Fatalf("container command = %#v, want /rstream-agent", container.Command)
	}
	if container.SecurityContext == nil || container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Fatalf("container security context does not disable privilege escalation")
	}
	if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
		t.Fatalf("container security context must use read-only root filesystem")
	}
	if got := container.Env[2].ValueFrom.SecretKeyRef.Name; got != "rstream-token" {
		t.Fatalf("token secret env name = %q, want rstream-token", got)
	}
	if got := dep.Spec.Template.Annotations[AnnotationConfigHash]; got == "" {
		t.Fatalf("missing config hash annotation")
	}
}

func TestApplyDeploymentMountsMTLSProjectedSecrets(t *testing.T) {
	tunnel := &tunnelsv1alpha1.RstreamTunnel{}
	tunnel.Name = "web"
	tunnel.Namespace = "demo"
	connection := &tunnelsv1alpha1.RstreamConnection{
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			Engine: "engine.example.com:443",
			MTLS: &tunnelsv1alpha1.MTLSAuthSpec{
				CertSecretRef: tunnelsv1alpha1.SecretKeyRef{Name: "agent-cert", Key: "tls.crt"},
				KeySecretRef:  tunnelsv1alpha1.SecretKeyRef{Name: "agent-key", Key: "tls.key"},
				CASecretRef:   &tunnelsv1alpha1.SecretKeyRef{Name: "engine-ca", Key: "ca.crt"},
			},
		},
	}
	var dep appsv1.Deployment
	err := ApplyDeployment(&dep, tunnel, connection, BuildOptions{AgentImage: "image", ConfigYAML: "config"})
	if err != nil {
		t.Fatalf("ApplyDeployment() error = %v", err)
	}
	var found bool
	for _, volume := range dep.Spec.Template.Spec.Volumes {
		if volume.Name == "mtls" {
			found = true
			if volume.Projected == nil || len(volume.Projected.Sources) != 3 {
				t.Fatalf("mtls projected sources = %#v, want 3 sources", volume.Projected)
			}
		}
	}
	if !found {
		t.Fatalf("missing mtls volume")
	}
	if dep.Spec.Template.Spec.AutomountServiceAccountToken == nil || !*dep.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatalf("agent must mount service account token to patch Tunnel status")
	}
}

func TestResolvedTargetAddress(t *testing.T) {
	target := ResolvedTarget{Host: "web.demo.svc", Port: 8080, Protocol: corev1.ProtocolTCP}
	if got := target.Address(); got != "web.demo.svc:8080" {
		t.Fatalf("Address() = %q", got)
	}
}

// See LICENSE file in the project root for license information.

package controller

import (
	"strings"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestResolveServiceTargetByName(t *testing.T) {
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		Spec: tunnelsv1alpha1.RstreamTunnelSpec{
			Protocol: tunnelsv1alpha1.ProtocolHTTP,
			Target: tunnelsv1alpha1.RstreamTunnelTarget{
				Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromString("http")},
			},
		},
	}
	svc := &corev1.Service{}
	svc.Name = "web"
	svc.Namespace = "demo"
	svc.Spec.Ports = []corev1.ServicePort{{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP}}
	target, err := resolveServiceTarget(tunnel, svc)
	if err != nil {
		t.Fatalf("resolveServiceTarget() error = %v", err)
	}
	if got := target.Address(); got != "web.demo.svc:8080" {
		t.Fatalf("Address() = %q", got)
	}
}

func TestResolveServiceTargetRejectsProtocolMismatch(t *testing.T) {
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		Spec: tunnelsv1alpha1.RstreamTunnelSpec{
			Protocol: tunnelsv1alpha1.ProtocolQUIC,
			Target: tunnelsv1alpha1.RstreamTunnelTarget{
				Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromInt(443)},
			},
		},
	}
	svc := &corev1.Service{}
	svc.Name = "web"
	svc.Namespace = "demo"
	svc.Spec.Ports = []corev1.ServicePort{{Name: "https", Port: 443, Protocol: corev1.ProtocolTCP}}
	_, err := resolveServiceTarget(tunnel, svc)
	if err == nil || !strings.Contains(err.Error(), "requires UDP") {
		t.Fatalf("resolveServiceTarget() error = %v, want UDP mismatch", err)
	}
}

// See LICENSE file in the project root for license information.

package agent

import (
	"context"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConnectingReusesOnlyTheConfirmedPublicHostname(t *testing.T) {
	const assigned = "stream.t.public.example"
	for _, test := range []struct {
		name   string
		status tunnelsv1alpha1.RstreamTunnelStatus
		want   string
	}{
		{name: "first connection"},
		{name: "legacy guess", status: tunnelsv1alpha1.RstreamTunnelStatus{Hostname: "stream.t.regional.example"}},
		{name: "legacy guess differs from last address", status: tunnelsv1alpha1.RstreamTunnelStatus{TunnelID: "old", Hostname: "stream.t.regional.example", ForwardingAddress: "https://" + assigned}},
		{name: "previous HTTP agent", status: tunnelsv1alpha1.RstreamTunnelStatus{TunnelID: "old", Hostname: assigned, ForwardingAddress: "https://" + assigned}, want: assigned},
		{name: "previous TLS agent", status: tunnelsv1alpha1.RstreamTunnelStatus{TunnelID: "old", Hostname: assigned, ForwardingAddress: assigned + " (tls)"}, want: assigned},
	} {
		t.Run(test.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := tunnelsv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			tunnel := &tunnelsv1alpha1.RstreamTunnel{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"}, Status: test.status}
			c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(tunnel).WithObjects(tunnel).Build()
			reporter := NewKubernetesReporter(c, agentconfig.KubernetesConfig{Enabled: true, Namespace: "demo", TunnelName: "web"})
			got, err := reporter.Connecting(context.Background())
			if err != nil || got != test.want {
				t.Fatalf("Connecting() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

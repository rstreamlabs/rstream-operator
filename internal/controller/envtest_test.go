// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestTunnelReconcileAgainstEnvtestAPI(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS is not set; run `make test` for envtest coverage")
	}
	ctx := context.Background()
	scheme := testScheme(t)
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}
	restConfig, err := testEnv.Start()
	if err != nil {
		t.Fatalf("envtest start failed: %v", err)
	}
	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("envtest stop failed: %v", err)
		}
	})
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("client.New() error = %v", err)
	}
	reconciler := &RstreamTunnelReconciler{Client: k8sClient, Scheme: scheme, AgentImage: "example.com/rstream-operator:test"}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "demo"}}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	connection := &tunnelsv1alpha1.RstreamConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "demo"},
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			Engine:         "project.c.example.com:443",
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{Name: "rstream-token", Key: "token"},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "rstream-token", Namespace: "demo"},
		Data:       map[string][]byte{"token": []byte("secret")},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP}},
		},
	}
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"},
		Spec: tunnelsv1alpha1.RstreamTunnelSpec{
			Target: tunnelsv1alpha1.RstreamTunnelTarget{
				Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromString("http")},
			},
			Protocol: tunnelsv1alpha1.ProtocolHTTP,
			HTTP:     &tunnelsv1alpha1.HTTPSpec{Version: tunnelsv1alpha1.HTTPVersion11},
		},
	}
	for _, obj := range []client.Object{connection, secret, svc, tunnel} {
		if err := k8sClient.Create(ctx, obj); err != nil {
			t.Fatalf("create %T: %v", obj, err)
		}
	}
	key := types.NamespacedName{Namespace: "demo", Name: "web"}
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	var updated tunnelsv1alpha1.RstreamTunnel
	if err := k8sClient.Get(ctx, key, &updated); err != nil {
		t.Fatalf("get RstreamTunnel: %v", err)
	}
	names := resources.NamesFor(&updated)
	var dep appsv1.Deployment
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: names.Deployment}, &dep); err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "example.com/rstream-operator:test" {
		t.Fatalf("deployment image = %q", dep.Spec.Template.Spec.Containers[0].Image)
	}
	if updated.Status.Target != "web.demo.svc:8080" {
		t.Fatalf("status target = %q", updated.Status.Target)
	}
}

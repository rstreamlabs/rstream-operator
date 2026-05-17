// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRstreamConnectionReconcilerMarksReady(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	connection := &tunnelsv1alpha1.RstreamConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "demo", Generation: 1},
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			Engine:         "engine.example.com:443",
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{Name: "rstream-token", Key: "token"},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "rstream-token", Namespace: "demo"},
		Data:       map[string][]byte{"token": []byte("value")},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tunnelsv1alpha1.RstreamTunnel{}, &tunnelsv1alpha1.RstreamConnection{}).
		WithObjects(connection, secret).
		Build()
	reconciler := &RstreamConnectionReconciler{Client: k8sClient, Scheme: scheme}
	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "default"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	var updated tunnelsv1alpha1.RstreamConnection
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: "default"}, &updated); err != nil {
		t.Fatalf("Connection get failed: %v", err)
	}
	if conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionReady) != metav1.ConditionTrue {
		t.Fatalf("Ready condition = %s, want True", conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionReady))
	}
}

func TestRstreamConnectionReconcilerResolvesProjectEndpoint(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	connection := &tunnelsv1alpha1.RstreamConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "demo", Generation: 1},
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			ProjectEndpoint: "abc12345",
			TokenSecretRef:  &tunnelsv1alpha1.SecretKeyRef{Name: "rstream-token", Key: "token"},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "rstream-token", Namespace: "demo"},
		Data:       map[string][]byte{"token": []byte("value")},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tunnelsv1alpha1.RstreamTunnel{}, &tunnelsv1alpha1.RstreamConnection{}).
		WithObjects(connection, secret).
		Build()
	reconciler := &RstreamConnectionReconciler{
		Client:             k8sClient,
		Scheme:             scheme,
		ConnectionResolver: fakeConnectionResolver{resolution: connectionResolution{Engine: "abc12345.c.rstream.io:443", APIURL: defaultAPIURL, ProjectID: "project-id", ProjectEndpoint: "abc12345"}},
	}
	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "default"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	var updated tunnelsv1alpha1.RstreamConnection
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: "default"}, &updated); err != nil {
		t.Fatalf("Connection get failed: %v", err)
	}
	if updated.Status.Engine != "abc12345.c.rstream.io:443" {
		t.Fatalf("status.engine = %q", updated.Status.Engine)
	}
	if updated.Status.ProjectEndpoint != "abc12345" || updated.Status.ProjectID != "project-id" {
		t.Fatalf("unexpected project status: %#v", updated.Status)
	}
}

type fakeConnectionResolver struct {
	resolution connectionResolution
	err        error
}

func (r fakeConnectionResolver) Resolve(context.Context, *tunnelsv1alpha1.RstreamConnection, string) (connectionResolution, error) {
	return r.resolution, r.err
}

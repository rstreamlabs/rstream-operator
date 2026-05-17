// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"strings"
	"testing"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTunnelReconcilerCreatesAgentResources(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		TypeMeta: metav1.TypeMeta{APIVersion: tunnelsv1alpha1.GroupVersion.String(), Kind: "RstreamTunnel"},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "web",
			Namespace:  "demo",
			Generation: 1,
			UID:        types.UID("uid-1"),
		},
		Spec: tunnelsv1alpha1.RstreamTunnelSpec{
			Target: tunnelsv1alpha1.RstreamTunnelTarget{
				Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromString("http")},
			},
			Protocol: tunnelsv1alpha1.ProtocolHTTP,
			HTTP: &tunnelsv1alpha1.HTTPSpec{
				Version: tunnelsv1alpha1.HTTPVersion11,
			},
		},
	}
	connection := &tunnelsv1alpha1.RstreamConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "demo", Generation: 1},
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			ProjectEndpoint: "abc12345",
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{
				Name: "rstream-token",
				Key:  "token",
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "rstream-token", Namespace: "demo", ResourceVersion: "1"},
		Data:       map[string][]byte{"token": []byte("secret")},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http", Port: 8080, Protocol: corev1.ProtocolTCP}},
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tunnelsv1alpha1.RstreamTunnel{}, &tunnelsv1alpha1.RstreamConnection{}).
		WithObjects(tunnel, connection, secret, svc).
		Build()
	reconciler := &RstreamTunnelReconciler{
		Client:             k8sClient,
		Scheme:             scheme,
		AgentImage:         "example.com/rstream-operator:test",
		ConnectionResolver: fakeConnectionResolver{resolution: connectionResolution{Engine: "abc12345.c.example.com:443", APIURL: defaultAPIURL, ProjectID: "project-id", ProjectEndpoint: "abc12345"}},
	}
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "web"}})
	if err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if !result.IsZero() {
		t.Fatalf("first Reconcile() result = %#v, want zero", result)
	}
	result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "web"}})
	if err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}
	if !result.IsZero() {
		t.Fatalf("second Reconcile() result = %#v, want zero", result)
	}
	names := resources.NamesFor(tunnel)
	var cm corev1.ConfigMap
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: names.ConfigMap}, &cm); err != nil {
		t.Fatalf("ConfigMap not created: %v", err)
	}
	if cm.Data[resources.ConfigFileName] == "" {
		t.Fatalf("agent config is empty")
	}
	if !strings.Contains(cm.Data[resources.ConfigFileName], "engine: abc12345.c.example.com:443") {
		t.Fatalf("agent config does not contain resolved engine:\n%s", cm.Data[resources.ConfigFileName])
	}
	var dep appsv1.Deployment
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: names.Deployment}, &dep); err != nil {
		t.Fatalf("Deployment not created: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "example.com/rstream-operator:test" {
		t.Fatalf("agent image = %q", dep.Spec.Template.Spec.Containers[0].Image)
	}
	if dep.Spec.Template.Spec.ServiceAccountName != names.ServiceAccount {
		t.Fatalf("service account = %q, want %q", dep.Spec.Template.Spec.ServiceAccountName, names.ServiceAccount)
	}
	var updated tunnelsv1alpha1.RstreamTunnel
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: "web"}, &updated); err != nil {
		t.Fatalf("Tunnel get failed: %v", err)
	}
	if updated.Status.Target != "web.demo.svc:8080" {
		t.Fatalf("status.target = %q", updated.Status.Target)
	}
	if updated.Status.Hostname == "" {
		t.Fatalf("status.hostname should contain generated stable hostname")
	}
	if conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionResolved) != metav1.ConditionTrue {
		t.Fatalf("Resolved condition = %s, want True", conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionResolved))
	}
}

func TestTunnelReconcilerMarksMissingSecret(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo", Generation: 1},
		Spec: tunnelsv1alpha1.RstreamTunnelSpec{
			Target: tunnelsv1alpha1.RstreamTunnelTarget{
				Service: tunnelsv1alpha1.ServiceTarget{Name: "web", Port: intstr.FromString("http")},
			},
		},
	}
	connection := &tunnelsv1alpha1.RstreamConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "demo", Generation: 1},
		Spec: tunnelsv1alpha1.RstreamConnectionSpec{
			Engine:         "project.c.example.com:443",
			TokenSecretRef: &tunnelsv1alpha1.SecretKeyRef{Name: "missing", Key: "token"},
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tunnelsv1alpha1.RstreamTunnel{}, &tunnelsv1alpha1.RstreamConnection{}).
		WithObjects(tunnel, connection).
		Build()
	reconciler := &RstreamTunnelReconciler{Client: k8sClient, Scheme: scheme, AgentImage: "image"}
	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "web"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "web"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	var updated tunnelsv1alpha1.RstreamTunnel
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: "web"}, &updated); err != nil {
		t.Fatalf("Tunnel get failed: %v", err)
	}
	if conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionReady) != metav1.ConditionFalse {
		t.Fatalf("Ready condition = %s, want False", conditionStatus(updated.Status.Conditions, tunnelsv1alpha1.ConditionReady))
	}
	if updated.Status.LastError == "" {
		t.Fatalf("LastError should explain missing secret")
	}
}

func TestTunnelReconcilerFinalizerDeletesChildren(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tunnel := &tunnelsv1alpha1.RstreamTunnel{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "web",
			Namespace:  "demo",
			UID:        types.UID("uid-1"),
			Finalizers: []string{tunnelFinalizer},
		},
	}
	now := metav1.Now()
	tunnel.DeletionTimestamp = &now
	names := resources.NamesFor(tunnel)
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tunnelsv1alpha1.RstreamTunnel{}, &tunnelsv1alpha1.RstreamConnection{}).
		WithObjects(
			tunnel,
			&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: names.Deployment, Namespace: "demo"}},
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: names.ConfigMap, Namespace: "demo"}},
		).
		Build()
	reconciler := &RstreamTunnelReconciler{Client: k8sClient, Scheme: scheme, AgentImage: "image"}
	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "demo", Name: "web"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	var dep appsv1.Deployment
	err = k8sClient.Get(ctx, types.NamespacedName{Namespace: "demo", Name: names.Deployment}, &dep)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Deployment get error = %v, want NotFound", err)
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(kubernetes) error = %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(apps) error = %v", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(rbac) error = %v", err)
	}
	if err := tunnelsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(rstream) error = %v", err)
	}
	return scheme
}

// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"fmt"
	"strings"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RstreamConnectionReconciler reconciles RstreamConnection resources.
type RstreamConnectionReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	ConnectionResolver connectionResolver
}

// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *RstreamConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var connection tunnelsv1alpha1.RstreamConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	refs := secretRefs(&connection)
	if len(refs) == 0 {
		return ctrl.Result{}, r.markConnection(ctx, req.NamespacedName, metav1.ConditionFalse, tunnelsv1alpha1.ReasonSecretMissing, "tokenSecretRef or mtls is required")
	}
	token := ""
	for _, ref := range refs {
		var secret corev1.Secret
		key := types.NamespacedName{Namespace: connection.Namespace, Name: ref.Name}
		if err := r.Get(ctx, key, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, r.markConnection(ctx, req.NamespacedName, metav1.ConditionFalse, tunnelsv1alpha1.ReasonSecretMissing, fmt.Sprintf("Secret %q was not found", ref.Name))
			}
			return ctrl.Result{}, err
		}
		if err := validateSecretKey(&secret, ref); err != nil {
			return ctrl.Result{}, r.markConnection(ctx, req.NamespacedName, metav1.ConditionFalse, tunnelsv1alpha1.ReasonSecretMissing, err.Error())
		}
		if connection.Spec.TokenSecretRef != nil && ref.Name == connection.Spec.TokenSecretRef.Name && ref.Key == connection.Spec.TokenSecretRef.Key {
			token = strings.TrimSpace(string(secret.Data[ref.Key]))
		}
	}
	resolution, err := r.connectionResolver().Resolve(ctx, &connection, token)
	if err != nil {
		return ctrl.Result{}, r.markConnection(ctx, req.NamespacedName, metav1.ConditionFalse, tunnelsv1alpha1.ReasonInvalidSpec, err.Error())
	}
	return ctrl.Result{}, r.markConnectionReady(ctx, req.NamespacedName, resolution)
}

func (r *RstreamConnectionReconciler) connectionResolver() connectionResolver {
	if r.ConnectionResolver != nil {
		return r.ConnectionResolver
	}
	return defaultConnectionResolver{}
}

func (r *RstreamConnectionReconciler) markConnection(ctx context.Context, key types.NamespacedName, status metav1.ConditionStatus, reason, message string) error {
	return patchConnectionStatus(ctx, r.Client, key, func(connection *tunnelsv1alpha1.RstreamConnection) {
		if status != metav1.ConditionTrue {
			connection.Status.APIURL = ""
			connection.Status.ProjectID = ""
			connection.Status.ProjectEndpoint = ""
			connection.Status.Engine = ""
		}
		setReady(&connection.Status.Conditions, connection.Generation, status, reason, message)
	})
}

func (r *RstreamConnectionReconciler) markConnectionReady(ctx context.Context, key types.NamespacedName, resolution connectionResolution) error {
	return patchConnectionStatus(ctx, r.Client, key, func(connection *tunnelsv1alpha1.RstreamConnection) {
		connection.Status.APIURL = resolution.APIURL
		connection.Status.ProjectID = resolution.ProjectID
		connection.Status.ProjectEndpoint = resolution.ProjectEndpoint
		connection.Status.Engine = resolution.Engine
		setReady(&connection.Status.Conditions, connection.Generation, metav1.ConditionTrue, tunnelsv1alpha1.ReasonReady, "Connection settings are ready.")
	})
}

func (r *RstreamConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tunnelsv1alpha1.RstreamConnection{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToConnections)).
		Complete(r)
}

func (r *RstreamConnectionReconciler) mapSecretToConnections(ctx context.Context, obj client.Object) []reconcile.Request {
	var connections tunnelsv1alpha1.RstreamConnectionList
	if err := r.List(ctx, &connections, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for _, connection := range connections.Items {
		if connectionReferencesSecret(&connection, obj.GetName()) {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: connection.Namespace, Name: connection.Name}})
		}
	}
	return requests
}

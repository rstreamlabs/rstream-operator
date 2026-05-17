// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rstreamlabs/rstream-go"
	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const tunnelFinalizer = "tunnels.rstream.io/finalizer"

// RstreamTunnelReconciler reconciles RstreamTunnel resources.
type RstreamTunnelReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	AgentImage         string
	ConnectionResolver connectionResolver
}

func (r *RstreamTunnelReconciler) connectionResolver() connectionResolver {
	if r.ConnectionResolver != nil {
		return r.ConnectionResolver
	}
	return defaultConnectionResolver{}
}

// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamtunnels,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamtunnels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamtunnels/finalizers,verbs=update
// +kubebuilder:rbac:groups=tunnels.rstream.io,resources=rstreamconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *RstreamTunnelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx).WithValues("rstreamtunnel", req.NamespacedName.String())
	var tunnel tunnelsv1alpha1.RstreamTunnel
	if err := r.Get(ctx, req.NamespacedName, &tunnel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !tunnel.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.finalize(ctx, &tunnel)
	}
	if !controllerutil.ContainsFinalizer(&tunnel, tunnelFinalizer) {
		controllerutil.AddFinalizer(&tunnel, tunnelFinalizer)
		if err := r.Update(ctx, &tunnel); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	resolvedConnection, err := r.resolveConnection(ctx, &tunnel)
	if err != nil {
		_ = r.markTunnelBlocked(ctx, req.NamespacedName, tunnelsv1alpha1.ReasonConnectionMissing, err.Error())
		return ctrl.Result{}, nil
	}
	target, err := r.resolveTarget(ctx, &tunnel)
	if err != nil {
		_ = r.markTargetBlocked(ctx, req.NamespacedName, err)
		return ctrl.Result{}, nil
	}
	hostname, err := r.desiredHostname(ctx, &tunnel, resolvedConnection.Engine)
	if err != nil {
		_ = r.markTunnelBlocked(ctx, req.NamespacedName, tunnelsv1alpha1.ReasonInvalidSpec, err.Error())
		return ctrl.Result{}, nil
	}
	agentConfig := resources.BuildAgentConfig(&tunnel, resolvedConnection.Connection, target, hostname, resolvedConnection.Engine)
	configYAML, err := resources.MarshalAgentConfig(agentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	opts := resources.BuildOptions{
		AgentImage:             r.AgentImage,
		ConfigYAML:             configYAML,
		SecretResourceVersions: resolvedConnection.SecretResourceVersions,
		Scheme:                 r.Scheme,
	}
	if err := r.applyChildren(ctx, &tunnel, resolvedConnection.Connection, opts); err != nil {
		return ctrl.Result{}, err
	}
	deploymentReady, err := r.deploymentReady(ctx, &tunnel)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.markReconciled(ctx, req.NamespacedName, &tunnel, target, hostname, deploymentReady); err != nil {
		return ctrl.Result{}, err
	}
	logger.V(1).Info("Tunnel reconciled")
	return ctrl.Result{}, nil
}

type resolvedConnection struct {
	Connection             *tunnelsv1alpha1.RstreamConnection
	Engine                 string
	APIURL                 string
	ProjectID              string
	ProjectEndpoint        string
	SecretResourceVersions []string
}

func (r *RstreamTunnelReconciler) resolveConnection(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel) (resolvedConnection, error) {
	var connection tunnelsv1alpha1.RstreamConnection
	key := types.NamespacedName{Namespace: tunnel.Namespace, Name: connectionName(tunnel)}
	if err := r.Get(ctx, key, &connection); err != nil {
		if apierrors.IsNotFound(err) {
			return resolvedConnection{}, fmt.Errorf("RstreamConnection %q was not found", key.Name)
		}
		return resolvedConnection{}, err
	}
	refs := secretRefs(&connection)
	if len(refs) == 0 {
		return resolvedConnection{}, fmt.Errorf("RstreamConnection %q must reference tokenSecretRef or mtls", key.Name)
	}
	versions := make([]string, 0, len(refs))
	token := ""
	for _, ref := range refs {
		var secret corev1.Secret
		secretKey := types.NamespacedName{Namespace: connection.Namespace, Name: ref.Name}
		if err := r.Get(ctx, secretKey, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				return resolvedConnection{}, fmt.Errorf("Secret %q was not found", ref.Name)
			}
			return resolvedConnection{}, err
		}
		if err := validateSecretKey(&secret, ref); err != nil {
			return resolvedConnection{}, err
		}
		if connection.Spec.TokenSecretRef != nil && ref.Name == connection.Spec.TokenSecretRef.Name && ref.Key == connection.Spec.TokenSecretRef.Key {
			token = strings.TrimSpace(string(secret.Data[ref.Key]))
		}
		versions = append(versions, ref.Name+":"+secret.ResourceVersion)
	}
	sort.Strings(versions)
	resolution, err := r.connectionResolver().Resolve(ctx, &connection, token)
	if err != nil {
		return resolvedConnection{}, err
	}
	return resolvedConnection{
		Connection:             &connection,
		Engine:                 resolution.Engine,
		APIURL:                 resolution.APIURL,
		ProjectID:              resolution.ProjectID,
		ProjectEndpoint:        resolution.ProjectEndpoint,
		SecretResourceVersions: versions,
	}, nil
}

func (r *RstreamTunnelReconciler) resolveTarget(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel) (resources.ResolvedTarget, error) {
	var svc corev1.Service
	key := types.NamespacedName{Namespace: serviceNamespace(tunnel), Name: tunnel.Spec.Target.Service.Name}
	if err := r.Get(ctx, key, &svc); err != nil {
		if apierrors.IsNotFound(err) {
			return resources.ResolvedTarget{}, fmt.Errorf("Service %q was not found in namespace %q", key.Name, key.Namespace)
		}
		return resources.ResolvedTarget{}, err
	}
	return resolveServiceTarget(tunnel, &svc)
}

func (r *RstreamTunnelReconciler) desiredHostname(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel, engine string) (string, error) {
	if tunnel.Spec.Publish != nil && !*tunnel.Spec.Publish {
		return "", nil
	}
	if strings.TrimSpace(tunnel.Spec.Hostname) != "" {
		return strings.TrimSpace(tunnel.Spec.Hostname), nil
	}
	if strings.TrimSpace(tunnel.Status.Hostname) != "" {
		return strings.TrimSpace(tunnel.Status.Hostname), nil
	}
	hostname, ok, err := rstream.GenerateStableDomain(engine)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	key := types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}
	if err := patchTunnelStatus(ctx, r.Client, key, func(current *tunnelsv1alpha1.RstreamTunnel) {
		current.Status.Hostname = hostname
	}); err != nil {
		return "", err
	}
	return hostname, nil
}

func (r *RstreamTunnelReconciler) applyChildren(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel, connection *tunnelsv1alpha1.RstreamConnection, opts resources.BuildOptions) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: resources.NamesFor(tunnel).ConfigMap, Namespace: tunnel.Namespace}}
	if _, err := createOrUpdate(ctx, r.Client, cm, func() error {
		return resources.ApplyConfigMap(cm, tunnel, opts)
	}); err != nil {
		return err
	}
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: resources.NamesFor(tunnel).ServiceAccount, Namespace: tunnel.Namespace}}
	if _, err := createOrUpdate(ctx, r.Client, sa, func() error {
		return resources.ApplyServiceAccount(sa, tunnel, opts)
	}); err != nil {
		return err
	}
	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: resources.NamesFor(tunnel).Role, Namespace: tunnel.Namespace}}
	if _, err := createOrUpdate(ctx, r.Client, role, func() error {
		return resources.ApplyRole(role, tunnel, opts)
	}); err != nil {
		return err
	}
	rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: resources.NamesFor(tunnel).RoleBinding, Namespace: tunnel.Namespace}}
	if _, err := createOrUpdate(ctx, r.Client, rb, func() error {
		return resources.ApplyRoleBinding(rb, tunnel, opts)
	}); err != nil {
		return err
	}
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: resources.NamesFor(tunnel).Deployment, Namespace: tunnel.Namespace}}
	if _, err := createOrUpdate(ctx, r.Client, dep, func() error {
		return resources.ApplyDeployment(dep, tunnel, connection, opts)
	}); err != nil {
		return err
	}
	return nil
}

func createOrUpdate(ctx context.Context, c client.Client, obj client.Object, mutate controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	var result controllerutil.OperationResult
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		result, err = controllerutil.CreateOrUpdate(ctx, c, obj, mutate)
		return err
	})
	return result, err
}

func (r *RstreamTunnelReconciler) deploymentReady(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel) (bool, error) {
	var dep appsv1.Deployment
	key := types.NamespacedName{Namespace: tunnel.Namespace, Name: resources.NamesFor(tunnel).Deployment}
	if err := r.Get(ctx, key, &dep); err != nil {
		return false, err
	}
	return dep.Status.AvailableReplicas >= 1 && dep.Status.ReadyReplicas >= 1, nil
}

func (r *RstreamTunnelReconciler) markTunnelBlocked(ctx context.Context, key types.NamespacedName, reason, message string) error {
	return patchTunnelStatus(ctx, r.Client, key, func(tunnel *tunnelsv1alpha1.RstreamTunnel) {
		tunnel.Status.LastError = message
		setCondition(&tunnel.Status.Conditions, tunnel.Generation, tunnelsv1alpha1.ConditionAccepted, metav1.ConditionFalse, reason, message)
		setReady(&tunnel.Status.Conditions, tunnel.Generation, metav1.ConditionFalse, reason, message)
	})
}

func (r *RstreamTunnelReconciler) markTargetBlocked(ctx context.Context, key types.NamespacedName, err error) error {
	message := err.Error()
	return patchTunnelStatus(ctx, r.Client, key, func(tunnel *tunnelsv1alpha1.RstreamTunnel) {
		tunnel.Status.LastError = message
		setCondition(&tunnel.Status.Conditions, tunnel.Generation, tunnelsv1alpha1.ConditionAccepted, metav1.ConditionTrue, tunnelsv1alpha1.ReasonAccepted, "Tunnel spec is accepted.")
		setCondition(&tunnel.Status.Conditions, tunnel.Generation, tunnelsv1alpha1.ConditionResolved, metav1.ConditionFalse, tunnelsv1alpha1.ReasonServiceMissing, message)
		setReady(&tunnel.Status.Conditions, tunnel.Generation, metav1.ConditionFalse, tunnelsv1alpha1.ReasonServiceMissing, message)
	})
}

func (r *RstreamTunnelReconciler) markReconciled(ctx context.Context, key types.NamespacedName, tunnel *tunnelsv1alpha1.RstreamTunnel, target resources.ResolvedTarget, hostname string, deploymentReady bool) error {
	names := resources.NamesFor(tunnel)
	return patchTunnelStatus(ctx, r.Client, key, func(current *tunnelsv1alpha1.RstreamTunnel) {
		current.Status.LastError = ""
		current.Status.AgentDeployment = names.Deployment
		current.Status.Target = target.Address()
		current.Status.RstreamName = resources.RstreamName(current)
		if hostname != "" {
			current.Status.Hostname = hostname
		}
		setCondition(&current.Status.Conditions, current.Generation, tunnelsv1alpha1.ConditionAccepted, metav1.ConditionTrue, tunnelsv1alpha1.ReasonAccepted, "Tunnel spec is accepted.")
		setCondition(&current.Status.Conditions, current.Generation, tunnelsv1alpha1.ConditionResolved, metav1.ConditionTrue, tunnelsv1alpha1.ReasonResolved, "Kubernetes target and rstream connection are resolved.")
		if deploymentReady {
			setCondition(&current.Status.Conditions, current.Generation, tunnelsv1alpha1.ConditionAgentReady, metav1.ConditionTrue, tunnelsv1alpha1.ReasonDeploymentAvailable, "Tunnel agent Deployment is available.")
			if conditionStatus(current.Status.Conditions, tunnelsv1alpha1.ConditionTunnelReady) == metav1.ConditionTrue {
				setReady(&current.Status.Conditions, current.Generation, metav1.ConditionTrue, tunnelsv1alpha1.ReasonReady, "Tunnel is ready.")
			} else {
				setReady(&current.Status.Conditions, current.Generation, metav1.ConditionFalse, tunnelsv1alpha1.ReasonWaitingForTunnel, "Waiting for the rstream tunnel to become ready.")
			}
		} else {
			setCondition(&current.Status.Conditions, current.Generation, tunnelsv1alpha1.ConditionAgentReady, metav1.ConditionFalse, tunnelsv1alpha1.ReasonDeploymentPending, "Tunnel agent Deployment is not available yet.")
			if conditionStatus(current.Status.Conditions, tunnelsv1alpha1.ConditionTunnelReady) == metav1.ConditionUnknown {
				setCondition(&current.Status.Conditions, current.Generation, tunnelsv1alpha1.ConditionTunnelReady, metav1.ConditionFalse, tunnelsv1alpha1.ReasonTunnelConnecting, "Tunnel agent has not reported readiness yet.")
			}
			setReady(&current.Status.Conditions, current.Generation, metav1.ConditionFalse, tunnelsv1alpha1.ReasonWaitingForDeployment, "Waiting for the tunnel agent Deployment to become available.")
		}
	})
}

func (r *RstreamTunnelReconciler) finalize(ctx context.Context, tunnel *tunnelsv1alpha1.RstreamTunnel) error {
	if !controllerutil.ContainsFinalizer(tunnel, tunnelFinalizer) {
		return nil
	}
	_ = patchTunnelStatus(ctx, r.Client, types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}, func(current *tunnelsv1alpha1.RstreamTunnel) {
		setReady(&current.Status.Conditions, current.Generation, metav1.ConditionFalse, tunnelsv1alpha1.ReasonFinalizing, "Deleting managed tunnel resources.")
	})
	names := resources.NamesFor(tunnel)
	for _, obj := range []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: names.Deployment, Namespace: tunnel.Namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: names.ConfigMap, Namespace: tunnel.Namespace}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: names.RoleBinding, Namespace: tunnel.Namespace}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: names.Role, Namespace: tunnel.Namespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: names.ServiceAccount, Namespace: tunnel.Namespace}},
	} {
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	controllerutil.RemoveFinalizer(tunnel, tunnelFinalizer)
	var latest tunnelsv1alpha1.RstreamTunnel
	if err := r.Get(ctx, types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}, &latest); err != nil {
		return client.IgnoreNotFound(err)
	}
	controllerutil.RemoveFinalizer(&latest, tunnelFinalizer)
	return r.Update(ctx, &latest)
}

func (r *RstreamTunnelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tunnelsv1alpha1.RstreamTunnel{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(&tunnelsv1alpha1.RstreamConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToTunnels)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.mapServiceToTunnels)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToTunnels)).
		Complete(r)
}

func (r *RstreamTunnelReconciler) mapConnectionToTunnels(ctx context.Context, obj client.Object) []reconcile.Request {
	var tunnels tunnelsv1alpha1.RstreamTunnelList
	if err := r.List(ctx, &tunnels, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for _, tunnel := range tunnels.Items {
		if connectionName(&tunnel) == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}})
		}
	}
	return requests
}

func (r *RstreamTunnelReconciler) mapServiceToTunnels(ctx context.Context, obj client.Object) []reconcile.Request {
	var tunnels tunnelsv1alpha1.RstreamTunnelList
	if err := r.List(ctx, &tunnels); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for _, tunnel := range tunnels.Items {
		targetNamespace := serviceNamespace(&tunnel)
		if targetNamespace == obj.GetNamespace() && tunnel.Spec.Target.Service.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}})
		}
	}
	return requests
}

func (r *RstreamTunnelReconciler) mapSecretToTunnels(ctx context.Context, obj client.Object) []reconcile.Request {
	var connections tunnelsv1alpha1.RstreamConnectionList
	if err := r.List(ctx, &connections, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	referencedConnections := map[string]struct{}{}
	for _, connection := range connections.Items {
		if connectionReferencesSecret(&connection, obj.GetName()) {
			referencedConnections[connection.Name] = struct{}{}
		}
	}
	if len(referencedConnections) == 0 {
		return nil
	}
	var tunnels tunnelsv1alpha1.RstreamTunnelList
	if err := r.List(ctx, &tunnels, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for _, tunnel := range tunnels.Items {
		if _, ok := referencedConnections[connectionName(&tunnel)]; ok {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: tunnel.Namespace, Name: tunnel.Name}})
		}
	}
	return requests
}

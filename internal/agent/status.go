// See LICENSE file in the project root for license information.

package agent

import (
	"context"
	"os"
	"strings"
	"time"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reporter interface {
	Connecting(context.Context) error
	Online(context.Context, OnlineStatus) error
	Disconnected(context.Context, error) error
}

type OnlineStatus struct {
	TunnelID          string
	Hostname          string
	ForwardingAddress string
	RstreamName       string
	Target            string
}

type NoopReporter struct{}

func (NoopReporter) Connecting(context.Context) error           { return nil }
func (NoopReporter) Online(context.Context, OnlineStatus) error { return nil }
func (NoopReporter) Disconnected(context.Context, error) error  { return nil }

type KubernetesReporter struct {
	client client.Client
	key    types.NamespacedName
	pod    string
}

func NewKubernetesReporter(c client.Client, cfg agentconfig.KubernetesConfig) Reporter {
	if !cfg.Enabled || c == nil {
		return NoopReporter{}
	}
	pod := ""
	if cfg.PodNameEnv != "" {
		pod = os.Getenv(cfg.PodNameEnv)
	}
	namespace := strings.TrimSpace(cfg.Namespace)
	if cfg.PodNamespaceEnv != "" {
		if envNamespace := strings.TrimSpace(os.Getenv(cfg.PodNamespaceEnv)); envNamespace != "" {
			namespace = envNamespace
		}
	}
	return &KubernetesReporter{
		client: c,
		key: types.NamespacedName{
			Namespace: namespace,
			Name:      cfg.TunnelName,
		},
		pod: pod,
	}
}

func (r *KubernetesReporter) Connecting(ctx context.Context) error {
	return r.patch(ctx, func(t *tunnelsv1alpha1.RstreamTunnel) {
		t.Status.LastError = ""
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionTunnelReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonTunnelConnecting,
			Message:            "Tunnel agent is connecting to rstream.",
		})
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonWaitingForTunnel,
			Message:            "Waiting for the rstream tunnel to become ready.",
		})
	})
}

func (r *KubernetesReporter) Online(ctx context.Context, status OnlineStatus) error {
	return r.patch(ctx, func(t *tunnelsv1alpha1.RstreamTunnel) {
		t.Status.TunnelID = status.TunnelID
		t.Status.Hostname = status.Hostname
		t.Status.ForwardingAddress = status.ForwardingAddress
		t.Status.RstreamName = status.RstreamName
		t.Status.Target = status.Target
		t.Status.LastError = ""
		message := "Tunnel is online."
		if r.pod != "" {
			message = "Tunnel is online from agent pod " + r.pod + "."
		}
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionTunnelReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonTunnelOnline,
			Message:            message,
		})
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonReady,
			Message:            "Tunnel is ready.",
		})
	})
}

func (r *KubernetesReporter) Disconnected(ctx context.Context, err error) error {
	message := "Tunnel is disconnected."
	if err != nil {
		message = err.Error()
	}
	return r.patch(ctx, func(t *tunnelsv1alpha1.RstreamTunnel) {
		t.Status.LastError = message
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionTunnelReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonTunnelDisconnected,
			Message:            message,
		})
		setCondition(&t.Status.Conditions, metav1.Condition{
			Type:               tunnelsv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: t.Generation,
			Reason:             tunnelsv1alpha1.ReasonWaitingForTunnel,
			Message:            "Waiting for the rstream tunnel to reconnect.",
		})
	})
}

func (r *KubernetesReporter) patch(ctx context.Context, mutate func(*tunnelsv1alpha1.RstreamTunnel)) error {
	if r == nil || r.client == nil {
		return nil
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var tunnel tunnelsv1alpha1.RstreamTunnel
		if err := r.client.Get(ctx, r.key, &tunnel); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		before := tunnel.DeepCopy()
		tunnel.Status.ObservedGeneration = tunnel.Generation
		mutate(&tunnel)
		for i := range tunnel.Status.Conditions {
			if tunnel.Status.Conditions[i].LastTransitionTime.IsZero() {
				tunnel.Status.Conditions[i].LastTransitionTime = metav1.NewTime(time.Now())
			}
		}
		return r.client.Status().Patch(ctx, &tunnel, client.MergeFrom(before))
	})
}

func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	meta.SetStatusCondition(conditions, condition)
}

// See LICENSE file in the project root for license information.

package controller

import (
	"context"
	"time"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func patchTunnelStatus(ctx context.Context, c client.Client, key types.NamespacedName, mutate func(*tunnelsv1alpha1.RstreamTunnel)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var tunnel tunnelsv1alpha1.RstreamTunnel
		if err := c.Get(ctx, key, &tunnel); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		before := tunnel.DeepCopy()
		tunnel.Status.ObservedGeneration = tunnel.Generation
		mutate(&tunnel)
		normalizeConditions(tunnel.Status.Conditions)
		return c.Status().Patch(ctx, &tunnel, client.MergeFrom(before))
	})
}

func patchConnectionStatus(ctx context.Context, c client.Client, key types.NamespacedName, mutate func(*tunnelsv1alpha1.RstreamConnection)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var connection tunnelsv1alpha1.RstreamConnection
		if err := c.Get(ctx, key, &connection); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		before := connection.DeepCopy()
		connection.Status.ObservedGeneration = connection.Generation
		mutate(&connection)
		normalizeConditions(connection.Status.Conditions)
		return c.Status().Patch(ctx, &connection, client.MergeFrom(before))
	})
}

func setReady(conditions *[]metav1.Condition, generation int64, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               tunnelsv1alpha1.ConditionReady,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

func setCondition(conditions *[]metav1.Condition, generation int64, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

func conditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {
	condition := meta.FindStatusCondition(conditions, conditionType)
	if condition == nil {
		return metav1.ConditionUnknown
	}
	return condition.Status
}

func normalizeConditions(conditions []metav1.Condition) {
	now := metav1.NewTime(time.Now())
	for i := range conditions {
		if conditions[i].LastTransitionTime.IsZero() {
			conditions[i].LastTransitionTime = now
		}
	}
}

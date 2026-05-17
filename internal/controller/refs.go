// See LICENSE file in the project root for license information.

package controller

import (
	"fmt"
	"sort"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const defaultConnectionName = "default"

type secretRef struct {
	Name string
	Key  string
}

func connectionName(tunnel *tunnelsv1alpha1.RstreamTunnel) string {
	if tunnel.Spec.ConnectionRef == nil || tunnel.Spec.ConnectionRef.Name == "" {
		return defaultConnectionName
	}
	return tunnel.Spec.ConnectionRef.Name
}

func secretRefs(connection *tunnelsv1alpha1.RstreamConnection) []secretRef {
	if connection == nil {
		return nil
	}
	refs := make([]secretRef, 0, 5)
	if connection.Spec.TokenSecretRef != nil {
		refs = append(refs, secretRef{Name: connection.Spec.TokenSecretRef.Name, Key: connection.Spec.TokenSecretRef.Key})
	}
	if connection.Spec.MTLS != nil {
		refs = append(refs,
			secretRef{Name: connection.Spec.MTLS.CertSecretRef.Name, Key: connection.Spec.MTLS.CertSecretRef.Key},
			secretRef{Name: connection.Spec.MTLS.KeySecretRef.Name, Key: connection.Spec.MTLS.KeySecretRef.Key},
		)
		if connection.Spec.MTLS.CASecretRef != nil {
			refs = append(refs, secretRef{Name: connection.Spec.MTLS.CASecretRef.Name, Key: connection.Spec.MTLS.CASecretRef.Key})
		}
	}
	if connection.Spec.Transport != nil && connection.Spec.Transport.Proxy != nil {
		if connection.Spec.Transport.Proxy.UsernameSecretRef != nil {
			refs = append(refs, secretRef{Name: connection.Spec.Transport.Proxy.UsernameSecretRef.Name, Key: connection.Spec.Transport.Proxy.UsernameSecretRef.Key})
		}
		if connection.Spec.Transport.Proxy.PasswordSecretRef != nil {
			refs = append(refs, secretRef{Name: connection.Spec.Transport.Proxy.PasswordSecretRef.Name, Key: connection.Spec.Transport.Proxy.PasswordSecretRef.Key})
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name == refs[j].Name {
			return refs[i].Key < refs[j].Key
		}
		return refs[i].Name < refs[j].Name
	})
	return refs
}

func connectionReferencesSecret(connection *tunnelsv1alpha1.RstreamConnection, secretName string) bool {
	for _, ref := range secretRefs(connection) {
		if ref.Name == secretName {
			return true
		}
	}
	return false
}

func validateSecretKey(secret *corev1.Secret, ref secretRef) error {
	if secret == nil {
		return fmt.Errorf("secret %q is missing", ref.Name)
	}
	if _, ok := secret.Data[ref.Key]; !ok {
		return fmt.Errorf("secret %q key %q is missing", ref.Name, ref.Key)
	}
	return nil
}

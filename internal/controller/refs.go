// See LICENSE file in the project root for license information.

package controller

import (
	"fmt"
	"sort"
	"strings"

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
	unique := make(map[secretRef]struct{}, 5+len(connection.Spec.ControlPlaneHeaders))
	for _, ref := range agentSecretRefs(connection) {
		unique[ref] = struct{}{}
	}
	for _, header := range connection.Spec.ControlPlaneHeaders {
		unique[secretRef{Name: header.ValueSecretRef.Name, Key: header.ValueSecretRef.Key}] = struct{}{}
	}
	return sortedSecretRefs(unique)
}

func agentSecretRefs(connection *tunnelsv1alpha1.RstreamConnection) []secretRef {
	if connection == nil {
		return nil
	}
	unique := make(map[secretRef]struct{}, 5)
	if connection.Spec.TokenSecretRef != nil {
		unique[secretRef{Name: connection.Spec.TokenSecretRef.Name, Key: connection.Spec.TokenSecretRef.Key}] = struct{}{}
	}
	if connection.Spec.MTLS != nil {
		unique[secretRef{Name: connection.Spec.MTLS.CertSecretRef.Name, Key: connection.Spec.MTLS.CertSecretRef.Key}] = struct{}{}
		unique[secretRef{Name: connection.Spec.MTLS.KeySecretRef.Name, Key: connection.Spec.MTLS.KeySecretRef.Key}] = struct{}{}
		if connection.Spec.MTLS.CASecretRef != nil {
			unique[secretRef{Name: connection.Spec.MTLS.CASecretRef.Name, Key: connection.Spec.MTLS.CASecretRef.Key}] = struct{}{}
		}
	}
	if connection.Spec.Transport != nil && connection.Spec.Transport.Proxy != nil {
		if connection.Spec.Transport.Proxy.UsernameSecretRef != nil {
			unique[secretRef{Name: connection.Spec.Transport.Proxy.UsernameSecretRef.Name, Key: connection.Spec.Transport.Proxy.UsernameSecretRef.Key}] = struct{}{}
		}
		if connection.Spec.Transport.Proxy.PasswordSecretRef != nil {
			unique[secretRef{Name: connection.Spec.Transport.Proxy.PasswordSecretRef.Name, Key: connection.Spec.Transport.Proxy.PasswordSecretRef.Key}] = struct{}{}
		}
	}
	return sortedSecretRefs(unique)
}

func sortedSecretRefs(unique map[secretRef]struct{}) []secretRef {
	refs := make([]secretRef, 0, len(unique))
	for ref := range unique {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name == refs[j].Name {
			return refs[i].Key < refs[j].Key
		}
		return refs[i].Name < refs[j].Name
	})
	return refs
}

func addControlPlaneHeaders(headers map[string]string, connection *tunnelsv1alpha1.RstreamConnection, ref secretRef, value string) {
	for _, header := range connection.Spec.ControlPlaneHeaders {
		if header.ValueSecretRef.Name == ref.Name && header.ValueSecretRef.Key == ref.Key {
			headers[header.Name] = strings.TrimSpace(value)
		}
	}
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

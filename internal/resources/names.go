// See LICENSE file in the project root for license information.

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	AppName = "rstream-operator"

	LabelName      = "app.kubernetes.io/name"
	LabelInstance  = "app.kubernetes.io/instance"
	LabelComponent = "app.kubernetes.io/component"
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelPartOf    = "app.kubernetes.io/part-of"

	LabelTunnelName      = "rstream.io/tunnel-name"
	LabelTunnelNamespace = "rstream.io/tunnel-namespace"
	LabelTunnelUID       = "rstream.io/tunnel-uid"

	AnnotationConfigHash = "rstream.io/config-hash"
)

type Names struct {
	Base           string
	ConfigMap      string
	Deployment     string
	ServiceAccount string
	Role           string
	RoleBinding    string
}

func NamesFor(tunnel *tunnelsv1alpha1.RstreamTunnel) Names {
	base := dnsLabel("rstream-" + tunnel.Name)
	hashInput := string(tunnel.UID)
	if strings.TrimSpace(hashInput) == "" {
		hashInput = tunnel.Namespace + "/" + tunnel.Name
	}
	hash := shortHash(hashInput)
	base = withHash(base, hash, 63)
	return Names{
		Base:           base,
		ConfigMap:      base + "-config",
		Deployment:     base,
		ServiceAccount: base,
		Role:           base,
		RoleBinding:    base,
	}
}

func ObjectKey(namespace string, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}

func AgentLabels(tunnel *tunnelsv1alpha1.RstreamTunnel, component string) map[string]string {
	names := NamesFor(tunnel)
	labels := map[string]string{
		LabelName:            AppName,
		LabelInstance:        names.Base,
		LabelComponent:       component,
		LabelManagedBy:       AppName,
		LabelPartOf:          "rstream",
		LabelTunnelName:      tunnel.Name,
		LabelTunnelNamespace: tunnel.Namespace,
	}
	if tunnel.UID != "" {
		labels[LabelTunnelUID] = string(tunnel.UID)
	}
	return labels
}

func ConfigHash(values ...string) string {
	sum := sha256.New()
	for _, value := range values {
		_, _ = sum.Write([]byte(value))
		_, _ = sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

func RstreamName(tunnel *tunnelsv1alpha1.RstreamTunnel) string {
	if strings.TrimSpace(tunnel.Spec.TunnelName) != "" {
		return strings.TrimSpace(tunnel.Spec.TunnelName)
	}
	return dnsLabel(fmt.Sprintf("%s-%s", tunnel.Namespace, tunnel.Name))
}

func dnsLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "rstream"
	}
	var b strings.Builder
	b.Grow(len(value))
	lastDash := false
	for _, r := range value {
		valid := unicode.IsLetter(r) || unicode.IsDigit(r)
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "rstream"
	}
	return out
}

func withHash(base, hash string, maxLen int) string {
	suffix := "-" + hash
	if len(base)+len(suffix) <= maxLen {
		return base + suffix
	}
	limit := maxLen - len(suffix)
	if limit < 1 {
		return hash[:maxLen]
	}
	return strings.TrimRight(base[:limit], "-") + suffix
}

func shortHash(value string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return fmt.Sprintf("%08x", h.Sum32())
}

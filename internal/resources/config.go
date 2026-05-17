// See LICENSE file in the project root for license information.

package resources

import (
	"fmt"
	"strconv"
	"strings"

	tunnelsv1alpha1 "github.com/rstreamlabs/rstream-operator/api/v1alpha1"
	"github.com/rstreamlabs/rstream-operator/internal/agentconfig"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	ConfigFileName = "config.yaml"

	AgentPodNameEnv      = "RSTREAM_AGENT_POD_NAME"
	AgentPodNamespaceEnv = "RSTREAM_AGENT_POD_NAMESPACE"
	ProxyUsernameEnv     = "RSTREAM_PROXY_USERNAME"
	ProxyPasswordEnv     = "RSTREAM_PROXY_PASSWORD"
)

type ResolvedTarget struct {
	Host     string
	Port     int32
	Protocol corev1.Protocol
}

func (t ResolvedTarget) Address() string {
	return t.Host + ":" + strconv.Itoa(int(t.Port))
}

func BuildAgentConfig(tunnel *tunnelsv1alpha1.RstreamTunnel, connection *tunnelsv1alpha1.RstreamConnection, target ResolvedTarget, hostname string, engine string) agentconfig.Config {
	publish := true
	if tunnel.Spec.Publish != nil {
		publish = *tunnel.Spec.Publish
	}
	tunnelType := inferTunnelType(tunnel)
	labels := copyMap(tunnel.Spec.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels["rstream.managed-by"] = "kubernetes-operator"
	labels["rstream.source"] = "kubernetes"
	labels["rstream.kubernetes.namespace"] = tunnel.Namespace
	labels["rstream.kubernetes.name"] = tunnel.Name
	if tunnel.UID != "" {
		labels["rstream.kubernetes.uid"] = string(tunnel.UID)
	}
	cfg := agentconfig.Config{
		Connection: agentconfig.ConnectionConfig{
			Engine:    strings.TrimSpace(engine),
			Transport: transportConfig(connection.Spec.Transport),
		},
		Tunnel: agentconfig.TunnelConfig{
			Name:        RstreamName(tunnel),
			Publish:     &publish,
			Protocol:    string(protocol(tunnel)),
			Type:        string(tunnelType),
			Hostname:    strings.TrimSpace(hostname),
			Labels:      labels,
			UpstreamTLS: tunnel.Spec.UpstreamTLS,
			TrustedIPs:  append([]string(nil), tunnel.Spec.TrustedIPs...),
			GeoIP:       append([]string(nil), tunnel.Spec.GeoIP...),
			HTTP:        httpConfig(tunnel.Spec.HTTP),
			TLS:         tlsConfig(tunnel.Spec.TLS),
		},
		Target: agentconfig.TargetConfig{
			Host:     target.Host,
			Port:     strconv.Itoa(int(target.Port)),
			Protocol: string(target.Protocol),
		},
		Kubernetes: agentconfig.KubernetesConfig{
			Enabled:         true,
			Namespace:       tunnel.Namespace,
			TunnelName:      tunnel.Name,
			PodNameEnv:      AgentPodNameEnv,
			PodNamespaceEnv: AgentPodNamespaceEnv,
		},
		Retry: agentconfig.RetryConfig{
			Initial: "1s",
			Max:     "30s",
		},
	}
	if connection.Spec.MTLS != nil {
		cfg.Connection.MTLS = &agentconfig.MTLSConfig{
			CertFile: "/var/run/rstream/mtls/tls.crt",
			KeyFile:  "/var/run/rstream/mtls/tls.key",
		}
		if connection.Spec.MTLS.CASecretRef != nil {
			cfg.Connection.MTLS.CAFile = "/var/run/rstream/mtls/ca.crt"
		}
	}
	return cfg
}

func MarshalAgentConfig(cfg agentconfig.Config) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal agent config: %w", err)
	}
	return string(data), nil
}

func protocol(tunnel *tunnelsv1alpha1.RstreamTunnel) tunnelsv1alpha1.Protocol {
	if tunnel.Spec.Protocol == "" {
		return tunnelsv1alpha1.ProtocolHTTP
	}
	return tunnel.Spec.Protocol
}

func inferTunnelType(tunnel *tunnelsv1alpha1.RstreamTunnel) tunnelsv1alpha1.TunnelType {
	if tunnel.Spec.Type != "" {
		return tunnel.Spec.Type
	}
	switch protocol(tunnel) {
	case tunnelsv1alpha1.ProtocolDTLS, tunnelsv1alpha1.ProtocolQUIC:
		return tunnelsv1alpha1.TunnelTypeDatagram
	case tunnelsv1alpha1.ProtocolHTTP:
		if tunnel.Spec.HTTP != nil && tunnel.Spec.HTTP.Version == tunnelsv1alpha1.HTTPVersion3 {
			return tunnelsv1alpha1.TunnelTypeDatagram
		}
	}
	return tunnelsv1alpha1.TunnelTypeBytestream
}

func httpConfig(spec *tunnelsv1alpha1.HTTPSpec) *agentconfig.HTTPConfig {
	if spec == nil {
		return nil
	}
	out := &agentconfig.HTTPConfig{Version: string(spec.Version)}
	if out.Version == "" {
		out.Version = string(tunnelsv1alpha1.HTTPVersion11)
	}
	if spec.Auth != nil {
		out.Auth = &agentconfig.HTTPAuthConfig{
			Token:   spec.Auth.Token,
			Rstream: spec.Auth.Rstream,
		}
	}
	if spec.Gate != nil {
		out.Gate = &agentconfig.HTTPGateConfig{Challenge: spec.Gate.Challenge}
	}
	return out
}

func tlsConfig(spec *tunnelsv1alpha1.TLSSpec) *agentconfig.TLSConfig {
	if spec == nil {
		return nil
	}
	return &agentconfig.TLSConfig{
		Mode:       string(spec.Mode),
		MinVersion: spec.MinVersion,
		ALPNs:      append([]string(nil), spec.ALPNs...),
		MTLS:       spec.MTLS,
	}
}

func transportConfig(spec *tunnelsv1alpha1.TransportSpec) *agentconfig.TransportConfig {
	if spec == nil {
		return nil
	}
	out := &agentconfig.TransportConfig{
		IPFamily: string(spec.IPFamily),
		MPTCP:    spec.MPTCP,
		UseQUIC:  spec.UseQUIC,
	}
	if spec.Bind != nil {
		out.Bind = &agentconfig.BindConfig{
			Mode:      spec.Bind.Mode,
			Interface: spec.Bind.Interface,
			Address:   spec.Bind.Address,
		}
	}
	if spec.DNS != nil {
		out.DNS = &agentconfig.DNSConfig{
			Override:   spec.DNS.Override,
			TLS:        spec.DNS.TLS,
			ServerName: spec.DNS.ServerName,
			DNSSEC:     spec.DNS.DNSSEC,
		}
	}
	if spec.Proxy != nil {
		out.Proxy = &agentconfig.ProxyConfig{
			HTTP:    spec.Proxy.HTTP,
			Headers: copyMap(spec.Proxy.Headers),
		}
		if spec.Proxy.UsernameSecretRef != nil {
			out.Proxy.UsernameEnv = ProxyUsernameEnv
		}
		if spec.Proxy.PasswordSecretRef != nil {
			out.Proxy.PasswordEnv = ProxyPasswordEnv
		}
	}
	return out
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

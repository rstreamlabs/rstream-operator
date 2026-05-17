// See LICENSE file in the project root for license information.

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// Protocol is the public protocol exposed by the rstream edge.
// +kubebuilder:validation:Enum=http;tls;dtls;quic
type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolTLS  Protocol = "tls"
	ProtocolDTLS Protocol = "dtls"
	ProtocolQUIC Protocol = "quic"
)

// TunnelType controls whether the tunnel carries stream or datagram traffic.
// +kubebuilder:validation:Enum=bytestream;datagram
type TunnelType string

const (
	TunnelTypeBytestream TunnelType = "bytestream"
	TunnelTypeDatagram   TunnelType = "datagram"
)

// HTTPVersion selects the HTTP version exposed by the rstream edge.
// +kubebuilder:validation:Enum=http/1.1;h2c;h3
type HTTPVersion string

const (
	HTTPVersion11 HTTPVersion = "http/1.1"
	HTTPVersion2  HTTPVersion = "h2c"
	HTTPVersion3  HTTPVersion = "h3"
)

// TLSMode selects how TLS is handled at the rstream edge.
// +kubebuilder:validation:Enum=terminated;passthrough
type TLSMode string

const (
	TLSModeTerminated  TLSMode = "terminated"
	TLSModePassthrough TLSMode = "passthrough"
)

// IPFamily selects the IP family used by the agent to reach the rstream engine.
// +kubebuilder:validation:Enum=auto;ipv4;ipv6
type IPFamily string

const (
	IPFamilyAuto IPFamily = "auto"
	IPFamilyIPv4 IPFamily = "ipv4"
	IPFamilyIPv6 IPFamily = "ipv6"
)

// SecretKeyRef references a single key in a Secret in the same namespace as the owning resource.
type SecretKeyRef struct {
	// Name is the Secret name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key is the key within the Secret data map.
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

// SecretKeySelector returns this reference as a Kubernetes SecretKeySelector.
func (r SecretKeyRef) SecretKeySelector() corev1.SecretKeySelector {
	return corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: r.Name},
		Key:                  r.Key,
	}
}

// TransportSpec configures the agent-to-engine network path.
type TransportSpec struct {
	// Bind controls the local bind address or interface used by the agent.
	// +optional
	Bind *BindSpec `json:"bind,omitempty"`
	// IPFamily forces IPv4 or IPv6. auto leaves selection to the host network stack.
	// +kubebuilder:default:=auto
	// +optional
	IPFamily IPFamily `json:"ipFamily,omitempty"`
	// DNS configures custom engine DNS resolution.
	// +optional
	DNS *DNSSpec `json:"dns,omitempty"`
	// MPTCP enables Multipath TCP when supported by the node kernel.
	// +optional
	MPTCP *bool `json:"mptcp,omitempty"`
	// Proxy configures an HTTP proxy for TLS engine transport.
	// +optional
	Proxy *ProxySpec `json:"proxy,omitempty"`
	// UseQUIC uses QUIC for the agent-to-engine transport.
	// +optional
	UseQUIC *bool `json:"useQuic,omitempty"`
}

// BindSpec controls local binding for engine connections.
type BindSpec struct {
	// Mode selects address or interface binding.
	// +kubebuilder:validation:Enum=address;interface
	// +kubebuilder:default:=address
	// +optional
	Mode string `json:"mode,omitempty"`
	// Interface is the network interface name when mode is interface.
	// +optional
	Interface string `json:"interface,omitempty"`
	// Address is the local source address when mode is address.
	// +optional
	Address string `json:"address,omitempty"`
}

// DNSSpec configures custom DNS resolution for the rstream engine.
type DNSSpec struct {
	// Override is the DNS server address to use for engine lookups.
	// +optional
	Override string `json:"override,omitempty"`
	// TLS enables DNS-over-TLS for the override resolver.
	// +optional
	TLS *bool `json:"tls,omitempty"`
	// ServerName is the DNS-over-TLS server name.
	// +optional
	ServerName string `json:"serverName,omitempty"`
	// DNSSEC enables DNSSEC validation where supported.
	// +optional
	DNSSEC *bool `json:"dnssec,omitempty"`
}

// ProxySpec configures an HTTP proxy for engine connections.
type ProxySpec struct {
	// HTTP is the proxy URL.
	// +optional
	HTTP string `json:"http,omitempty"`
	// UsernameSecretRef references the proxy username.
	// +optional
	UsernameSecretRef *SecretKeyRef `json:"usernameSecretRef,omitempty"`
	// PasswordSecretRef references the proxy password.
	// +optional
	PasswordSecretRef *SecretKeyRef `json:"passwordSecretRef,omitempty"`
	// Headers are additional HTTP headers sent to the proxy.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`
}

// See LICENSE file in the project root for license information.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// RstreamTunnelSpec defines the desired state of a rstream tunnel.
type RstreamTunnelSpec struct {
	// ConnectionRef references the rstream connection settings in this namespace.
	// Defaults to a RstreamConnection named default.
	// +optional
	ConnectionRef *corev1.LocalObjectReference `json:"connectionRef,omitempty"`
	// Target is the Kubernetes backend reached by the tunnel agent.
	Target RstreamTunnelTarget `json:"target"`
	// TunnelName overrides the name registered in rstream. By default the operator uses namespace-name.
	// +optional
	TunnelName string `json:"tunnelName,omitempty"`
	// Publish exposes the tunnel through the rstream edge.
	// +kubebuilder:default:=true
	// +optional
	Publish *bool `json:"publish,omitempty"`
	// Protocol is the public protocol exposed by the rstream edge.
	// +kubebuilder:default:=http
	// +optional
	Protocol Protocol `json:"protocol,omitempty"`
	// Type controls bytestream or datagram forwarding. When omitted the operator infers it from protocol/http.version.
	// +optional
	Type TunnelType `json:"type,omitempty"`
	// Hostname sets a stable public hostname. When omitted, the operator generates one once and stores it in status.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Labels are attached to the rstream tunnel inventory object.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// UpstreamTLS tells the edge that the Kubernetes upstream service expects TLS.
	// +optional
	UpstreamTLS *bool `json:"upstreamTLS,omitempty"`
	// TrustedIPs restricts access to the published tunnel by source CIDR.
	// +optional
	TrustedIPs []string `json:"trustedIPs,omitempty"`
	// GeoIP restricts access to the published tunnel by country code.
	// +optional
	GeoIP []string `json:"geoip,omitempty"`
	// HTTP configures HTTP-specific edge behavior.
	// +optional
	HTTP *HTTPSpec `json:"http,omitempty"`
	// TLS configures TLS-specific edge behavior.
	// +optional
	TLS *TLSSpec `json:"tls,omitempty"`
	// Agent customizes the managed data-plane Deployment.
	// +optional
	Agent *AgentSpec `json:"agent,omitempty"`
}

// RstreamTunnelTarget identifies the Kubernetes backend reached by the tunnel agent.
type RstreamTunnelTarget struct {
	// Service targets a Kubernetes Service.
	Service ServiceTarget `json:"service"`
}

// ServiceTarget identifies a Kubernetes Service port.
type ServiceTarget struct {
	// Namespace is the Service namespace. Defaults to the RstreamTunnel namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Name is the Service name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Port is the Service port name or number.
	Port intstr.IntOrString `json:"port"`
}

// HTTPSpec configures HTTP tunnel behavior.
type HTTPSpec struct {
	// Version selects the HTTP version exposed by the rstream edge.
	// +kubebuilder:default:=http/1.1
	// +optional
	Version HTTPVersion `json:"version,omitempty"`
	// Auth configures edge authentication.
	// +optional
	Auth *HTTPAuthSpec `json:"auth,omitempty"`
	// Gate configures browser challenge behavior.
	// +optional
	Gate *HTTPGateSpec `json:"gate,omitempty"`
}

// HTTPAuthSpec configures HTTP edge authentication.
type HTTPAuthSpec struct {
	// Token enables token authentication at the edge.
	// +optional
	Token *bool `json:"token,omitempty"`
	// Rstream enables rstream account authentication at the edge.
	// +optional
	Rstream *bool `json:"rstream,omitempty"`
}

// HTTPGateSpec configures HTTP browser challenge behavior.
type HTTPGateSpec struct {
	// Challenge enables edge challenge mode.
	// +optional
	Challenge *bool `json:"challenge,omitempty"`
}

// TLSSpec configures TLS tunnel behavior.
type TLSSpec struct {
	// Mode selects TLS termination behavior at the edge.
	// +optional
	Mode TLSMode `json:"mode,omitempty"`
	// MinVersion is the minimum TLS version accepted by the edge.
	// +kubebuilder:validation:Enum=tls1.2;tls1.3
	// +optional
	MinVersion string `json:"minVersion,omitempty"`
	// ALPNs configures allowed application protocols.
	// +optional
	ALPNs []string `json:"alpns,omitempty"`
	// MTLS enables client mTLS authentication at the edge.
	// +optional
	MTLS *bool `json:"mtls,omitempty"`
}

// AgentSpec customizes the managed data-plane Deployment.
type AgentSpec struct {
	// Image overrides the tunnel agent image. Defaults to the controller --agent-image flag.
	// +optional
	Image string `json:"image,omitempty"`
	// Replicas is intentionally limited to 1 until multi-agent tunnel ownership is explicitly supported.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1
	// +kubebuilder:default:=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Resources configures CPU and memory requests/limits for the agent container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// PodLabels are merged into the managed Pod template labels.
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`
	// PodAnnotations are merged into the managed Pod template annotations.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// NodeSelector constrains where the agent Pod can run.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Tolerations configures Pod tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Affinity configures Pod affinity.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// PriorityClassName sets the Pod priority class.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// RstreamTunnelStatus defines the observed state of RstreamTunnel.
type RstreamTunnelStatus struct {
	// ObservedGeneration is the latest metadata.generation reconciled by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions describe reconciliation, agent readiness, and tunnel connectivity.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// TunnelID is the rstream tunnel identifier reported by the agent.
	// +optional
	TunnelID string `json:"tunnelID,omitempty"`
	// RstreamName is the name registered in rstream.
	// +optional
	RstreamName string `json:"rstreamName,omitempty"`
	// Hostname is the stable hostname requested for published tunnels.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// ForwardingAddress is the public address returned by rstream for published tunnels.
	// +optional
	ForwardingAddress string `json:"forwardingAddress,omitempty"`
	// Target is the resolved upstream host:port dialed by the agent.
	// +optional
	Target string `json:"target,omitempty"`
	// AgentDeployment is the managed Deployment name.
	// +optional
	AgentDeployment string `json:"agentDeployment,omitempty"`
	// LastError contains the latest human-readable error reported by the controller or agent.
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rstreamtunnels,singular=rstreamtunnel,scope=Namespaced,categories=rstream,shortName=tunnel;rtun;rtunnel
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.status.target`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.forwardingAddress`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RstreamTunnel declares a managed rstream tunnel to a Kubernetes Service.
type RstreamTunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RstreamTunnelSpec   `json:"spec,omitempty"`
	Status RstreamTunnelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RstreamTunnelList contains a list of RstreamTunnel.
type RstreamTunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RstreamTunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RstreamTunnel{}, &RstreamTunnelList{})
}

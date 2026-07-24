// See LICENSE file in the project root for license information.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RstreamConnectionSpec defines shared rstream connection settings.
//
// +kubebuilder:validation:XValidation:rule="has(self.tokenSecretRef) != has(self.mtls)",message="exactly one of tokenSecretRef or mtls must be set"
// +kubebuilder:validation:XValidation:rule="(has(self.projectEndpoint) ? 1 : 0) + (has(self.projectID) ? 1 : 0) + (has(self.engine) ? 1 : 0) == 1",message="exactly one of projectEndpoint, projectID, or engine must be set"
// +kubebuilder:validation:XValidation:rule="has(self.engine) || has(self.tokenSecretRef)",message="projectEndpoint and projectID require tokenSecretRef"
// +kubebuilder:validation:XValidation:rule="!has(self.engine) || !has(self.region)",message="region cannot be used with an explicit engine"
// +kubebuilder:validation:XValidation:rule="!has(self.engine) || !has(self.controlPlaneHeaders) || size(self.controlPlaneHeaders) == 0",message="controlPlaneHeaders cannot be used with an explicit engine"
type RstreamConnectionSpec struct {
	// ProjectEndpoint identifies the rstream project to resolve through the Control plane. This is the preferred hosted rstream path.
	// +kubebuilder:validation:MinLength=1
	// +optional
	ProjectEndpoint string `json:"projectEndpoint,omitempty"`
	// ProjectID identifies the rstream project to resolve through the Control plane when the endpoint is not known.
	// +kubebuilder:validation:MinLength=1
	// +optional
	ProjectID string `json:"projectID,omitempty"`
	// APIURL is the rstream Control plane URL used for project resolution.
	// +kubebuilder:default:="https://rstream.io"
	// +kubebuilder:validation:MinLength=1
	// +optional
	APIURL string `json:"apiURL,omitempty"`
	// Region selects an authorized regional endpoint. When omitted or set to auto, global projects use their global endpoint and regional projects use their assigned region.
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$`
	// +optional
	Region string `json:"region,omitempty"`
	// ControlPlaneHeaders adds deployment-specific HTTP headers to Control plane requests. Values are read from Secrets and are never forwarded to engine endpoints.
	// +listType=map
	// +listMapKey=name
	// +optional
	ControlPlaneHeaders []ControlPlaneHeader `json:"controlPlaneHeaders,omitempty"`
	// Engine is the rstream engine address, for example engine.rstream.io:443. Use this for self-hosted deployments without a Control plane.
	// +kubebuilder:validation:MinLength=1
	// +optional
	Engine string `json:"engine,omitempty"`
	// TokenSecretRef references the bearer token used by tunnel agents.
	// +optional
	TokenSecretRef *SecretKeyRef `json:"tokenSecretRef,omitempty"`
	// MTLS configures client certificate authentication for tunnel agents.
	// +optional
	MTLS *MTLSAuthSpec `json:"mtls,omitempty"`
	// Transport configures the agent-to-engine network path.
	// +optional
	Transport *TransportSpec `json:"transport,omitempty"`
}

// ControlPlaneHeader references a secret value sent as an HTTP header to the Control plane.
type ControlPlaneHeader struct {
	// Name is the HTTP header name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Name string `json:"name"`
	// ValueSecretRef references the header value.
	ValueSecretRef SecretKeyRef `json:"valueSecretRef"`
}

// MTLSAuthSpec references the client certificate material used for rstream agent authentication.
type MTLSAuthSpec struct {
	// CertSecretRef references the PEM client certificate.
	CertSecretRef SecretKeyRef `json:"certSecretRef"`
	// KeySecretRef references the PEM client private key.
	KeySecretRef SecretKeyRef `json:"keySecretRef"`
	// CASecretRef optionally references a PEM CA bundle for the engine TLS connection.
	// +optional
	CASecretRef *SecretKeyRef `json:"caSecretRef,omitempty"`
}

// RstreamConnectionStatus defines the observed state of RstreamConnection.
type RstreamConnectionStatus struct {
	// ObservedGeneration is the latest metadata.generation reconciled by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions describe the current connection configuration state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// APIURL is the Control plane URL used for the last successful resolution.
	// +optional
	APIURL string `json:"apiURL,omitempty"`
	// ProjectID is the resolved project ID when project lookup is used.
	// +optional
	ProjectID string `json:"projectID,omitempty"`
	// ProjectEndpoint is the resolved project endpoint when project lookup is used.
	// +optional
	ProjectEndpoint string `json:"projectEndpoint,omitempty"`
	// Region is the endpoint selection used for the last successful resolution.
	// +optional
	Region string `json:"region,omitempty"`
	// Engine is the resolved engine address used by tunnel agents.
	// +optional
	Engine string `json:"engine,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=rstream,shortName=rconn
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.status.projectEndpoint`
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.status.engine`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RstreamConnection provides shared rstream connection settings for RstreamTunnel resources.
type RstreamConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RstreamConnectionSpec   `json:"spec,omitempty"`
	Status RstreamConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RstreamConnectionList contains a list of RstreamConnection.
type RstreamConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RstreamConnection `json:"items"`
}

func addRstreamConnectionTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &RstreamConnection{}, &RstreamConnectionList{})
	return nil
}

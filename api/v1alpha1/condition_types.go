// See LICENSE file in the project root for license information.

package v1alpha1

const (
	// ConditionReady reports the user-facing readiness of the resource.
	ConditionReady = "Ready"
	// ConditionAccepted reports whether the spec is syntactically and semantically accepted.
	ConditionAccepted = "Accepted"
	// ConditionResolved reports whether Kubernetes references have been resolved.
	ConditionResolved = "Resolved"
	// ConditionAgentReady reports whether the managed tunnel-agent Deployment is available.
	ConditionAgentReady = "AgentReady"
	// ConditionTunnelReady reports whether the rstream control channel has an active tunnel.
	ConditionTunnelReady = "TunnelReady"
)

const (
	ReasonAccepted             = "Accepted"
	ReasonInvalidSpec          = "InvalidSpec"
	ReasonConnectionMissing    = "ConnectionMissing"
	ReasonSecretMissing        = "SecretMissing"
	ReasonServiceMissing       = "ServiceMissing"
	ReasonServicePortMissing   = "ServicePortMissing"
	ReasonResolved             = "Resolved"
	ReasonReconciling          = "Reconciling"
	ReasonDeploymentAvailable  = "DeploymentAvailable"
	ReasonDeploymentPending    = "DeploymentPending"
	ReasonTunnelConnecting     = "TunnelConnecting"
	ReasonTunnelOnline         = "TunnelOnline"
	ReasonTunnelDisconnected   = "TunnelDisconnected"
	ReasonFinalizing           = "Finalizing"
	ReasonReady                = "Ready"
	ReasonWaitingForTunnel     = "WaitingForTunnel"
	ReasonWaitingForDeployment = "WaitingForDeployment"
)

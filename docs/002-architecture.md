# Architecture and credentials

The manager watches the Kubernetes API and reconciles `RstreamConnection`
and `RstreamTunnel` resources. Each tunnel gets a dedicated agent. Requests
reach the Engine through its published ingress and leave the agent through
the target Kubernetes Service.

| Component | Network access |
| --- | --- |
| Manager | Kubernetes API and, for hosted projects, the configured Control plane URL. |
| Agent Pod | Kubernetes API for status, cluster DNS, the Engine using the selected transport, and the target Service and port. |
| Published caller | The Engine's published address for the specified protocol. |

Allow these paths when writing NetworkPolicies. The agent opens the
Engine connection outbound. The target Service can remain a ClusterIP
Service. For Engine transport selection, see the [API reference](003-api-reference.md#transport).

## Kubernetes resource ownership

`RstreamConnection` and its credential Secrets live in the tunnel namespace.
Multiple tunnels can reference one connection there. A target Service uses the
tunnel namespace by default. To target a Service elsewhere, set
`spec.target.service.namespace` and allow that cross-namespace traffic.

The manager makes one Deployment, ConfigMap, ServiceAccount, Role, and
RoleBinding for each tunnel. Kubernetes owner references tie them to the
`RstreamTunnel`. Change the tunnel spec to customize its agent and leave the
managed children to the controller. The operator uses one replica and the
Recreate strategy for the agent, so replacing it can interrupt forwarding.

Delete a tunnel while the manager is running. The finalizer removes the
managed resources before allowing the custom resource to disappear. The
application Service, connection, and credential Secrets have separate
lifecycles.

## Credential access

| Credential | Who uses it |
| --- | --- |
| Project or Engine token | The agent reads it from a Secret-backed environment variable. The manager also uses it for hosted project lookup. |
| mTLS certificate, private key, and optional CA | The agent reads projected Secret files. |
| Proxy username and password | The agent reads Secret-backed environment variables. |
| Control plane access-gateway headers | The manager reads their Secret keys and sends them only to the Control plane during lookup. |

The Helm manager has cluster-wide read access to Secrets. Limit which
principals can create or change connection and tunnel resources, and who
can access the managed Pods. The agent ServiceAccount role can read its own
tunnel and write its status. It has no Secret API permissions. Kubernetes
injects credentials when starting the Pod.

The agent ConfigMap contains endpoints and policy. Secret values stay in
Secrets. Inline settings such as `transport.proxy.headers` can appear in the
ConfigMap, so use the supported Secret references for proxy credentials
and access-gateway headers. The agent runs as non-root with a read-only
root filesystem and dropped capabilities.

## Readiness and recovery

| Status | Writer and meaning |
| --- | --- |
| Connection `Ready` | The manager reports whether credential references and connection settings resolve. |
| Tunnel `Accepted`, `Resolved`, `AgentReady` | The manager reports whether it can use the spec and references, resolve the Service, and bring up the agent Deployment. |
| Tunnel `TunnelReady`, `tunnelID`, `hostname`, `forwardingAddress` | The agent reports online state and the resulting tunnel ID, hostname, and address. The manager initializes `TunnelReady` while waiting for the agent's first report. |
| Tunnel `Ready` | Both processes update the aggregate condition as the Deployment and forwarding session change. |
| Tunnel `lastError` | Both processes use this field for their latest reconciliation or runtime error. |

The agent marks the tunnel unready and retries with backoff when its
Engine connection drops. Status can retain the last reported forwarding
address while the agent reconnects. Missing references block updates to
the managed agent. An agent already running can keep its previous
configuration until reconciliation succeeds or the tunnel is deleted.

Check the conditions and make a real request to verify that a tunnel and
its upstream are reachable. See [Troubleshooting](005-troubleshooting.md).

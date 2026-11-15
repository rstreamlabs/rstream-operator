# Troubleshooting

The tunnel status shows whether a failure comes from the specification, a referenced resource, the agent Pod or the connection to rstream. Start with:

```sh
kubectl -n <namespace> describe rstreamtunnel <name>
```

## Ready is False

Check the conditions in order:

| Condition | Meaning |
| --- | --- |
| `Accepted=False` | The spec is invalid. |
| `Resolved=False` | A referenced Service, port, connection or Secret is missing. |
| `AgentReady=False` | Kubernetes has not made the agent Deployment available. |
| `TunnelReady=False` | The agent cannot connect to rstream or create the tunnel. |

## Secret Missing

The Secret must be in the same namespace as the `RstreamConnection`. Inspect the key names and sizes without printing credential values:

```sh
kubectl -n <namespace> describe secret <name>
```

## Service Port Missing

Use the Service port, not the Pod container port:

```sh
kubectl -n <namespace> get service <service> -o yaml
```

Named ports are preferred because they survive port number changes.

## Project Resolution Fails

When `RstreamConnection.spec.projectEndpoint` or `spec.projectID` is used, the manager calls the Control plane before it creates the agent config.

| Check | Expected result |
| --- | --- |
| `spec.apiURL` | Reachable from the manager Pod; defaults to `https://rstream.io`. |
| `tokenSecretRef` | References a token permitted to read the project. |
| `spec.region` | `auto` or a region authorized for the project. |
| `controlPlaneHeaders.valueSecretRef` | Each referenced Secret contains the expected access-gateway value. |
| `kubectl describe rstreamconnection <name>` | Shows `Ready=True` and `status.engine`. |

`status.region` reports `auto` or the explicit region used by the last successful resolution. Access-gateway headers are manager-only credentials and must not appear in the generated agent ConfigMap.

Self-hosted deployments without a Control plane should use `spec.engine` directly.

## Protocol Mismatch

Match the Kubernetes Service transport to the tunnel protocol:

| Tunnel configuration | Service transport |
| --- | --- |
| `protocol: quic` or `protocol: dtls` | UDP |
| `protocol: http` with `http.version: h3` | UDP |
| `protocol: http` with `http/1.1` or `h2c` | TCP |
| `protocol: tls` | TCP |

## Agent Logs

```sh
deployment="$(kubectl -n <namespace> get rstreamtunnel <name> -o jsonpath='{.status.agentDeployment}')"
kubectl -n <namespace> logs "deployment/${deployment}"
```

Look for:

- `connect control channel`
- `create tunnel`
- `Tunnel online`
- reconnect warnings

## Forwarding Address Empty

`status.forwardingAddress` is reported by the agent after it successfully creates the rstream tunnel. If Kubernetes resources are ready but this field is empty, inspect agent logs and the `TunnelReady` condition.

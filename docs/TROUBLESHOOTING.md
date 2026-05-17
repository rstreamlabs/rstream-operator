# Troubleshooting

Start with:

```sh
kubectl -n <namespace> describe rstreamtunnel <name>
```

## Ready is False

Check the conditions in order:

- `Accepted=False`: the spec is invalid.
- `Resolved=False`: a referenced Service, port, connection, or Secret is missing.
- `AgentReady=False`: Kubernetes has not made the agent Deployment available.
- `TunnelReady=False`: the agent cannot connect to rstream or cannot create the tunnel.

## Secret Missing

The Secret must be in the same namespace as the `RstreamConnection`.

```sh
kubectl -n <namespace> get secret <name> -o jsonpath='{.data}'
```

## Service Port Missing

Use the Service port, not the Pod container port:

```sh
kubectl -n <namespace> get service <service> -o yaml
```

Named ports are preferred because they survive port number changes.

## Project Resolution Fails

When `RstreamConnection.spec.projectEndpoint` or `spec.projectID` is used, the manager calls the Control plane before it creates the agent config.

Check:

- `spec.apiURL` is reachable from the manager Pod. It defaults to `https://rstream.io`.
- `tokenSecretRef` points to a token that can read the project.
- `kubectl describe rstreamconnection <name>` shows `Ready=True` and `status.engine`.

Self-hosted deployments without a Control plane should use `spec.engine` directly.

## Protocol Mismatch

UDP tunnels need a UDP Service port:

- `protocol: quic`
- `protocol: dtls`
- `protocol: http` with `http.version: h3`

TCP tunnels need a TCP Service port:

- `protocol: http` with `http/1.1` or `h2c`
- `protocol: tls`

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

# Troubleshoot a tunnel

Follow the path from the Kubernetes API to the application. The commands
below use the release and resource names from [the first tunnel](001-first-tunnel.md).

## Check the manager and resource status

```sh
kubectl api-resources --api-group=tunnels.rstream.io
kubectl -n rstream-system rollout status deployment/rstream-operator --timeout=30s
kubectl -n rstream-demo describe rstreamconnection default
kubectl -n rstream-demo describe rstreamtunnel web
```

If the API is missing, check the Helm release and whether the CRDs were
installed. Inspect the manager logs if the resources exist but their
conditions are blank.

```sh
kubectl -n rstream-system logs deployment/rstream-operator --tail=100
```

Use the current condition reason and message. Compare its
`observedGeneration` with `metadata.generation` if you recently edited the
resource.

| Condition or reason | What to check |
| --- | --- |
| Connection `Ready=False`, `SecretMissing` | The Secret and key must exist in the connection namespace. Use `kubectl describe secret` to verify key names and sizes without printing values. |
| Connection `Ready=False`, `InvalidSpec` | Read the message for project lookup errors. Verify the endpoint or ID, token access, authorized region, and manager connectivity to `apiURL`. For a direct Engine, set `spec.engine` and omit the hosted lookup settings. |
| Tunnel `Accepted=False`, `InvalidSpec` | Compare the tunnel settings with the [API rules](003-api-reference.md#rstreamtunnel). An API admission error can reject the resource before a condition is created. |
| Tunnel `Accepted=False`, `ConnectionMissing` | Check the same-namespace connection and its status. Secret or lookup errors for the connection can surface under this reason. |
| Tunnel `Resolved=False`, `ServiceMissing` | Check the target Service namespace, port name or number, and TCP or UDP protocol. This reason also covers incompatible or missing Service ports. |
| Tunnel `AgentReady=False` or `TunnelReady=False` | Check the Deployment and Pod events, then read agent logs. The agent becomes ready after the Engine accepts the tunnel. |

The agent may report `TunnelReady=True` before Kubernetes has marked
the Deployment available. For a new tunnel, wait for `AgentReady=True`
and then `Ready=True`.

After fixing a temporary project lookup failure, update the connection
to trigger reconciliation. An annotation works when the spec can stay
the same.

```sh
kubectl -n rstream-demo annotate rstreamconnection default --overwrite \
  "rstream.io/reconcile-at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Follow the agent

```sh
deployment="$(kubectl -n rstream-demo get rstreamtunnel web -o jsonpath='{.status.agentDeployment}')"
kubectl -n rstream-demo describe "deployment/$deployment"
kubectl -n rstream-demo get pods -l rstream.io/tunnel-name=web
kubectl -n rstream-demo logs "deployment/$deployment" -c agent --tail=100
```

If `status.agentDeployment` is empty, return to the tunnel conditions.
Otherwise, inspect the Pod events for scheduling, image pulls, or
Secret mounts. The logs distinguish `connect control channel` from
`create tunnel`. Connection errors point to DNS, egress, certificate
trust, or agent authentication. Creation errors indicate that the
request reached the Engine. Check the reported policy, protocol, hostname,
or quota error.

## Check a published address

```sh
address="$(kubectl -n rstream-demo get rstreamtunnel web -o jsonpath='{.status.forwardingAddress}')"
curl --verbose --fail --show-error "$address"
kubectl -n rstream-demo get endpointslice -l kubernetes.io/service-name=http-server
```

| Symptom | Where to look |
| --- | --- |
| Public address is missing | Confirm `spec.publish` is true and `TunnelReady=True`. A value such as `rstrm://rstream-demo-web (unpublished)` identifies a private tunnel. |
| DNS or certificate validation fails | Check that the address resolves to the Engine's published ingress and the certificate covers the returned hostname. |
| HTTP authentication or policy error | Check the tunnel's edge access policy and send any required caller credential. |
| Tunnel is online but requests time out or fail at the upstream | Confirm the Service has ready EndpointSlices, selects the intended Pods, and allows connections from the agent. Test the Service directly from inside the cluster. |

Use an HTTP client for HTTP tunnels. For another protocol, use a client that
speaks that protocol and matches the tunnel's access policy. Refer to the
[API reference](003-api-reference.md#protocol-and-service-port-rules) for
the Service protocol mapping.

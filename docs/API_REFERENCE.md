# API Reference

This document describes the `tunnels.rstream.io/v1alpha1` API.

## RstreamConnection

`RstreamConnection` is namespaced. It stores shared project, engine, and credential settings used by `RstreamTunnel` resources in the same namespace.

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
spec:
  projectEndpoint: "<project-endpoint>"
  tokenSecretRef:
    name: rstream-credentials
    key: token
```

### Spec

| Field | Required | Description |
| --- | --- | --- |
| `projectEndpoint` | one of project endpoint, project ID, or engine | rstream project endpoint resolved through the Control plane. This is the preferred hosted path. |
| `projectID` | one of project endpoint, project ID, or engine | rstream project ID resolved through the Control plane. Use it when the endpoint is not known. |
| `apiURL` | no | Control plane URL used for project lookup. Defaults to `https://rstream.io`. |
| `engine` | one of project endpoint, project ID, or engine | rstream engine address, usually `host:port`. Use it for self-hosted deployments without a Control plane. |
| `tokenSecretRef` | one of token or mTLS | Secret key containing the bearer token. |
| `mtls` | one of token or mTLS | Secret references for client certificate authentication. |
| `transport` | no | Advanced agent-to-engine network settings. |

Exactly one of `projectEndpoint`, `projectID`, or `engine` must be set. `projectEndpoint` and `projectID` require `tokenSecretRef` because the operator resolves the engine through the Control plane. Direct `engine` connections can use either token or mTLS authentication.

`tokenSecretRef` and `mtls` are mutually exclusive.

### Direct engine address

For self-hosted engines or internal environments without a Control plane, set `engine` directly:

```yaml
spec:
  engine: engine.rstream.io:443
  tokenSecretRef:
    name: rstream-credentials
    key: token
```

### mTLS

```yaml
spec:
  engine: engine.rstream.io:443
  mtls:
    certSecretRef:
      name: rstream-agent-mtls
      key: tls.crt
    keySecretRef:
      name: rstream-agent-mtls
      key: tls.key
    caSecretRef:
      name: rstream-engine-ca
      key: ca.crt
```

`caSecretRef` is optional and should be used when the engine certificate chain is not trusted by the default system roots.

### Transport

```yaml
spec:
  transport:
    ipFamily: ipv4
    useQuic: false
    dns:
      override: 1.1.1.1
      tls: true
      serverName: cloudflare-dns.com
    proxy:
      http: http://proxy.local:3128
      usernameSecretRef:
        name: proxy-credentials
        key: username
      passwordSecretRef:
        name: proxy-credentials
        key: password
```

Use transport settings only when required by the network environment. The default path is usually best.

## RstreamTunnel

`RstreamTunnel` is namespaced. It declares one rstream tunnel to a Kubernetes Service.

### Minimal HTTP RstreamTunnel

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
spec:
  target:
    service:
      name: http-server
      port: http
  publish: true
  protocol: http
  http:
    version: http/1.1
```

### Spec

| Field | Required | Description |
| --- | --- | --- |
| `connectionRef.name` | no | `RstreamConnection` name. Defaults to `default`. |
| `target.service.name` | yes | Kubernetes Service name. |
| `target.service.namespace` | no | Target namespace. Defaults to the RstreamTunnel namespace. |
| `target.service.port` | yes | Service port name or number. |
| `tunnelName` | no | Name registered in rstream. Defaults to `<namespace>-<name>`. |
| `publish` | no | Publish through the rstream edge. Defaults to `true`. |
| `protocol` | no | `http`, `tls`, `dtls`, or `quic`. Defaults to `http`. |
| `type` | no | `bytestream` or `datagram`. Inferred when omitted. |
| `hostname` | no | Stable public hostname. Generated and persisted in status when omitted. |
| `labels` | no | Labels added to the rstream tunnel inventory. |
| `upstreamTLS` | no | Marks the Kubernetes upstream as TLS-enabled. |
| `trustedIPs` | no | Source CIDRs allowed by the edge. |
| `geoip` | no | Country codes allowed by the edge. |
| `http` | no | HTTP-specific options. |
| `tls` | no | TLS-specific options. |
| `agent` | no | Managed data-plane Pod customization. |

### Protocol and Service Port Rules

The operator validates that the Service port protocol matches the tunnel data type:

- `http` with `http/1.1` or `h2c`: Service port must be TCP.
- `http` with `h3`: Service port must be UDP.
- `tls`: Service port must be TCP.
- `dtls` or `quic`: Service port must be UDP.

### HTTP

```yaml
http:
  version: http/1.1
  auth:
    token: true
    rstream: true
  gate:
    challenge: false
```

`version` can be `http/1.1`, `h2c`, or `h3`.

### TLS

```yaml
protocol: tls
tls:
  mode: terminated
  minVersion: tls1.3
  alpns:
    - h2
  mtls: true
```

`mode` can be `terminated` or `passthrough`.

### Agent

```yaml
agent:
  image: rstream/rstream-operator:latest
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      memory: 512Mi
  nodeSelector:
    kubernetes.io/os: linux
  tolerations: []
  podLabels:
    team: platform
```

`agent.replicas` is constrained to `1` for now. This avoids ambiguous ownership of a single tunnel hostname until multi-agent semantics are explicitly guaranteed by the rstream engine.

## Status

`RstreamConnection.status` includes the resolved project and engine fields:

| Field | Description |
| --- | --- |
| `conditions` | Connection configuration readiness. |
| `apiURL` | Control plane URL used for project resolution. |
| `projectID` | Resolved project ID when lookup is used. |
| `projectEndpoint` | Resolved project endpoint when lookup is used. |
| `engine` | Engine address used by tunnel agents. |

`RstreamTunnel.status` includes:

| Field | Description |
| --- | --- |
| `observedGeneration` | Last spec generation reconciled. |
| `conditions` | Readiness and diagnostic conditions. |
| `tunnelID` | rstream tunnel ID reported by the agent. |
| `rstreamName` | Name registered in rstream. |
| `hostname` | Stable hostname requested for the tunnel. |
| `forwardingAddress` | Public forwarding address returned by rstream. |
| `target` | Resolved upstream `host:port`. |
| `agentDeployment` | Managed Deployment name. |
| `lastError` | Latest controller or agent error. |

Condition types:

- `Accepted`
- `Resolved`
- `AgentReady`
- `TunnelReady`
- `Ready`

# API reference

Both resources use `tunnels.rstream.io/v1alpha1`. Use
`kubectl explain rstreamconnection.spec` and
`kubectl explain rstreamtunnel.spec` for the schema installed in your
cluster. See [the first tunnel](001-first-tunnel.md) for a complete example.

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
| `region` | no | Authorized regional endpoint selected for the agent control channel. Omit it or use `auto` for the project default. |
| `controlPlaneHeaders` | no | Additional Control plane HTTP headers whose values come from Secret keys. |
| `engine` | one of project endpoint, project ID, or engine | rstream engine address, usually `host:port`. Use it for self-hosted deployments without a Control plane. |
| `tokenSecretRef` | one of token or mTLS | Secret key containing the bearer token. |
| `mtls` | one of token or mTLS | Secret references for client certificate authentication. |
| `transport` | no | Advanced agent-to-engine network settings. |

Exactly one of `projectEndpoint`, `projectID`, or `engine` must be set. `projectEndpoint` and `projectID` require `tokenSecretRef` because the operator resolves the engine through the Control plane. Direct `engine` connections can use either token or mTLS authentication.

`tokenSecretRef` and `mtls` are mutually exclusive.

### Region selection

A Global project uses its global endpoint when `region` is omitted or set
to `auto`. Set an explicit region from the project's authorized endpoints
to keep the agent control channel there.

```yaml
spec:
  projectEndpoint: "<project-endpoint>"
  region: us-east-1
```

Use region selection with a hosted project lookup.

### Control plane headers

Reference the access-gateway credentials in a Secret when a private
Control plane uses them.

```yaml
spec:
  projectEndpoint: "<project-endpoint>"
  controlPlaneHeaders:
    - name: x-deployment-bypass
      valueSecretRef:
        name: rstream-control-plane
        key: deployment-bypass
```

The manager sends these headers only while resolving the project through
`apiURL`. The SDK rejects reserved authentication and forwarding headers.

### Direct engine address

Specify the Engine address directly for self-hosted deployments.

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

`caSecretRef` is optional. When set, it supplies the trust roots for the
agent's Engine connection. With token authentication, install any private
Engine CA in the agent image's system trust store. See how to
[select an agent image](004-installation-reference.md#helm-values).

### Transport

```yaml
spec:
  transport:
    mode: tls
    ipFamily: ipv4
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

`mode` accepts `auto`, `tls`, or `quic`. The default `auto` mode prefers
QUIC and falls back to TLS while opening the agent control channel, then
keeps the selected transport for that agent client. Use `mode` by itself
in new resources. Older manifests can still use `useQuic`.

The transport fields apply to the agent-to-engine connection, independently of
`RstreamTunnel.spec.protocol`, which selects the public tunnel protocol.

| Field under `transport` | Contract |
| --- | --- |
| `mode` | `auto`, `tls`, or `quic`. Omit for `auto`, or specify legacy `useQuic` alone. |
| `ipFamily` | `auto`, `ipv4` or `ipv6`; defaults to the host network stack with `auto`. |
| `bind.mode` | `address` or `interface`; defaults to `address`. |
| `bind.address`, `bind.interface` | Local source address or interface for the selected binding mode. |
| `dns.override` | Custom resolver address for engine lookups. |
| `dns.tls`, `dns.serverName` | DNS-over-TLS selection and certificate server name for that resolver. |
| `dns.dnssec` | Request DNSSEC validation through the agent SDK. |
| `mptcp` | Enable Multipath TCP where the node kernel supports it. |
| `proxy.http` | HTTP proxy URL for TLS engine transport. |
| `proxy.usernameSecretRef`, `proxy.passwordSecretRef` | Same-namespace Secret key references for proxy credentials. |
| `proxy.headers` | Inline proxy headers. These values are visible in the generated agent ConfigMap. |
| `useQuic` | Legacy boolean transport selector. |

The published tunnel's access policy is configured on the tunnel. Keep
credentials in Secret references.

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
| `publish` | no | Publish through the rstream edge. Defaults to `true`. Set `false` for a tunnel reachable through authenticated rstream clients. |
| `protocol` | no | `http`, `tls`, `dtls`, `quic`, or `tcp`. Defaults to `http`. |
| `type` | no | `bytestream` or `datagram`. Inferred when omitted. |
| `datagramGuaranteedDelivery` | no | Requires reliable delivery for datagram tunnels. Defaults to `false`. |
| `allowCrossRegionRouting` | no | Permits a cross-region payload path when ingress and tunnel owner are in different regions. Same-region traffic remains direct. |
| `hostname` | no | Request a specific published hostname. When omitted, the agent records the Engine-confirmed hostname in status and can reuse it on reconnect. |
| `tcpPort` | no | Reserved public port requested by a published TCP tunnel. Omit it for an ephemeral port. |
| `labels` | no | Labels added to the rstream tunnel inventory. |
| `upstreamTLS` | no | Marks the Kubernetes upstream as TLS-enabled. |
| `trustedIPs` | no | Source CIDRs allowed by the edge. |
| `geoip` | no | Country codes allowed by the edge. |
| `http` | no | HTTP-specific options. |
| `tls` | no | TLS-specific options. |
| `agent` | no | Managed data-plane Pod customization. |

### Protocol and Service Port Rules

Set `target.service.port` to a Service port name or its `spec.ports[].port`
number. The operator infers the Service port protocol from the tunnel
protocol when `type` is omitted.

| Tunnel protocol | Service port protocol |
| --- | --- |
| `http` with `http/1.1` or `h2c` | TCP |
| `http` with `h3` | UDP |
| `tls` | TCP |
| `tcp` | TCP. Use a published bytestream tunnel. |
| `dtls` or `quic` | UDP |

An explicit `type: datagram` requires a UDP Service port. An explicit
`type: bytestream` requires TCP. The Engine also validates whether the
requested type and protocol can be combined.

Set `datagramGuaranteedDelivery: true` only on datagram tunnels. It disables unreliable datagram fast paths, including QUIC datagrams on the agent-to-Engine leg, when packet loss is unacceptable.

Set `allowCrossRegionRouting: true` when the deployment policy permits a
cross-region payload path. When omitted or false, the Engine uses Direct
routing. With the option enabled, same-region traffic still uses the direct path.

### Published TCP

```yaml
protocol: tcp
tcpPort: 10042
```

Omit `tcpPort` to let the Engine allocate an ephemeral port. Reserve any
explicit port for the project through the rstream CLI or Dashboard first.
Published TCP carries application bytes as-is. Use an encrypted application
protocol such as SSH, or choose a TLS tunnel for TLS termination or passthrough.

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

| HTTP field | Purpose |
| --- | --- |
| `version` | `http/1.1`, `h2c`, or `h3`. The default is `http/1.1`. |
| `auth.token` | Enable token authentication for callers at the edge. |
| `auth.rstream` | Enable rstream account authentication for callers at the edge. |
| `gate.challenge` | Enable the edge browser challenge. |
| `cors` | Configure allowed browser origins and cross-origin requests. |

The HTTP authentication settings apply to callers. Configure the agent's
own Engine credentials on `RstreamConnection`.

### HTTP CORS

Set `spec.http.cors` to configure a published HTTP tunnel's CORS policy.

```yaml
http:
  auth:
    token: true
  cors:
    allowedOrigins: [https://dashboard.example]
    allowedMethods: [GET, PUT]
    allowedHeaders: [Authorization, Content-Type]
    exposedHeaders: [ETag]
    allowCredentials: false
    maxAge: 600
```

CORS requires a published HTTP tunnel and at least one explicit HTTP(S)
origin without paths or wildcards. Each list accepts at most 128 entries.
The Engine limits all strings together to 16 KiB. `maxAge` is 0–86400
seconds. Omitted methods default to GET, HEAD, and POST. Omitted request
headers default to Authorization and Content-Type. With no policy, the
upstream handles CORS. The edge handles preflights and applies
authentication to actual requests.

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

| TLS field | Purpose |
| --- | --- |
| `mode` | Select `terminated` or `passthrough` at the edge. |
| `minVersion` | Minimum edge TLS version, `tls1.2` or `tls1.3`. |
| `alpns` | Allowed application protocols. |
| `mtls` | Enable caller mTLS at the edge. |

### Agent

```yaml
agent:
  image: rstream/rstream-operator:0.6.2
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

| Agent field | Purpose and default |
| --- | --- |
| `image` | Override the manager's configured default agent image. |
| `replicas` | One agent per tunnel. The only accepted value is `1`. |
| `resources` | Pod resources. When omitted, the agent requests 25m CPU and 64Mi memory with a 256Mi memory limit. |
| `nodeSelector`, `tolerations`, `affinity`, `priorityClassName` | Agent scheduling settings. |
| `podLabels`, `podAnnotations` | Agent Pod metadata. |

## Status

### RstreamConnection

Read `status.engine` to see which Engine the manager resolved.

| Field | Description |
| --- | --- |
| `observedGeneration` | Last connection spec generation reconciled. |
| `conditions` | Connection configuration readiness. |
| `apiURL` | Control plane URL used for project resolution. |
| `projectID` | Resolved project ID when lookup is used. |
| `projectEndpoint` | Resolved project endpoint when lookup is used. |
| `engine` | Engine address used by tunnel agents. |
| `region` | `auto` or the explicit region for a resolved hosted project. |

### RstreamTunnel

| Field | Description |
| --- | --- |
| `observedGeneration` | Last spec generation reconciled. |
| `conditions` | Readiness and diagnostic conditions. |
| `tunnelID` | rstream tunnel ID reported by the agent. |
| `rstreamName` | Name registered in rstream. |
| `hostname` | The Engine-confirmed hostname for a published tunnel. |
| `forwardingAddress` | Display address reported by the agent. Published tunnels show their public address. Private tunnels can show `rstrm://<tunnel-name> (unpublished)`. The field can retain its last value during reconnect. |
| `target` | Resolved upstream `host:port`. |
| `agentDeployment` | Managed Deployment name. |
| `lastError` | Latest controller or agent error. |

See [status ownership](002-architecture.md#readiness-and-recovery) for
the condition types and [troubleshooting](005-troubleshooting.md) for
their reasons and messages.

# rstream-operator

`rstream-operator` manages rstream tunnels from Kubernetes. It lets teams expose a Kubernetes Service through rstream with a small declarative API instead of running tunnel commands manually.

The operator is built with standard controller-runtime/Kubebuilder patterns:

- `RstreamConnection` stores shared project, engine, and credential settings.
- `RstreamTunnel` declares one managed tunnel to a Kubernetes `Service`.
- The controller creates a dedicated tunnel-agent `Deployment` per `RstreamTunnel`.
- The tunnel agent owns the data plane and reports tunnel readiness back into `RstreamTunnel.status`.

Release images are published on Docker Hub as `rstream/rstream-operator`.

## Status

The API is currently `v1alpha1`. Treat field names and semantics as production-quality, but not yet immutable.

## Quick Start

Install the operator:

```sh
helm upgrade --install rstream-operator ./charts/rstream-operator \
  --namespace rstream-system \
  --create-namespace
```

Create an application Service:

```sh
kubectl apply -f config/samples/http_server.yaml
```

Create a Secret containing a rstream token with access to the project:

```sh
kubectl -n rstream-demo create secret generic rstream-credentials \
  --from-literal=token="$RSTREAM_TOKEN"
```

Create the rstream connection and tunnel:

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: rstream-demo
spec:
  projectEndpoint: "<project-endpoint>"
  tokenSecretRef:
    name: rstream-credentials
    key: token
---
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
  namespace: rstream-demo
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

Check readiness and the public address:

```sh
kubectl -n rstream-demo get rstreamtunnel web
kubectl -n rstream-demo describe rstreamtunnel web
```

## API Shape

Use `RstreamConnection` when multiple tunnels share the same rstream project and credentials:

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: platform
spec:
  projectEndpoint: "<project-endpoint>"
  apiURL: https://rstream.io
  tokenSecretRef:
    name: rstream-credentials
    key: token
```

`projectEndpoint` is the preferred hosted rstream path. The operator resolves the current engine address through the Control plane and stores the result in `RstreamConnection.status.engine`.

For self-hosted engines or internal test environments without a Control plane, specify `engine` directly instead:

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: platform
spec:
  engine: engine.internal.example.com:443
  tokenSecretRef:
    name: rstream-credentials
    key: token
```

Use `RstreamTunnel` to expose a Service:

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
  namespace: platform
spec:
  connectionRef:
    name: default
  target:
    service:
      name: web
      port: http
  publish: true
  protocol: http
  labels:
    app: web
  http:
    version: http/1.1
    auth:
      token: true
      rstream: true
```

See [docs/API_REFERENCE.md](docs/API_REFERENCE.md) for the field reference.

## Runtime Model

The controller does not proxy application traffic. For each `RstreamTunnel`, it creates:

- a `ConfigMap` containing non-secret agent configuration;
- a dedicated `ServiceAccount`, `Role`, and `RoleBinding`;
- a single-replica `Deployment` running `/rstream-agent`;
- a restricted Pod security context and read-only root filesystem.

The agent reads credentials from Kubernetes Secrets at runtime, opens an outbound control channel to the rstream engine, forwards traffic to the target Service, and patches `RstreamTunnel.status`.

This model keeps traffic isolation simple: restarting or deleting one `RstreamTunnel` only impacts its own agent Pod.

## Installation Options

Helm:

```sh
helm upgrade --install rstream-operator ./charts/rstream-operator \
  --namespace rstream-system \
  --create-namespace \
  --set image.repository=rstream/rstream-operator \
  --set image.tag=latest
```

Kustomize:

```sh
make install
make deploy IMAGE=rstream/rstream-operator:latest
```

`AGENT_IMAGE` defaults to `IMAGE` for `make deploy`; set it explicitly only when agents should run a different image.

For local development:

```sh
make run AGENT_IMAGE=rstream/rstream-operator:latest
```

## Observability

`kubectl get rstreamtunnel` shows readiness, protocol, target, and forwarding address. The CRD also exposes `tunnel`, `rtun`, and `rtunnel` as short aliases for interactive use.

`kubectl describe rstreamtunnel <name>` shows conditions:

- `Accepted`: the `RstreamTunnel` spec is valid.
- `Resolved`: referenced Service, port, connection, and Secret keys exist.
- `AgentReady`: the managed Deployment is available.
- `TunnelReady`: the agent has an active rstream tunnel.
- `Ready`: aggregate user-facing readiness.

Agent logs are JSON and include tunnel ID, forwarding address, target, and reconnect errors.

## Security

Credentials stay in Kubernetes Secrets and are not copied into ConfigMaps.

The managed agent Pod runs non-root, drops Linux capabilities, disables privilege escalation, uses the runtime-default seccomp profile, and has a read-only root filesystem.

The per-tunnel agent Role is limited to:

- `get` its own `RstreamTunnel`;
- `get`, `patch`, and `update` its own `RstreamTunnel/status`.

The manager needs cluster-wide watch access because `RstreamTunnel` can target Services in another namespace. If you do not need cross-namespace targets, keep `spec.target.service.namespace` unset.

## Development

The repository keeps tools local under `bin/`.

```sh
make generate
make manifests
make test
make helm-lint
make build
```

For local development next to a sibling `../rstream-go` checkout, create an uncommitted Go workspace:

```sh
go work init . ../rstream-go
```

`go.mod` pins the SDK as a normal module dependency so release builds do not depend on local filesystem layout.

Release builds use the public module metadata from `go.mod` and do not require GitHub SSH credentials. The runtime smoke script automatically uses the sibling `../rstream-go` checkout when present, which keeps unreleased SDK changes testable before a module tag is cut. Set `RSTREAM_GO_REPO=/path/to/rstream-go` when your local SDK checkout lives somewhere else.

## Runtime Smoke Test

With Docker, Kind, kubectl, Helm, and a usable rstream token:

```sh
export RSTREAM_PROJECT_ENDPOINT="project-endpoint"
export RSTREAM_TOKEN=...
hack/runtime-smoke.sh
```

Use `RSTREAM_ENGINE=...` instead of `RSTREAM_PROJECT_ENDPOINT` when testing a self-hosted or internal engine without a Control plane.

The script builds the image, loads it into Kind, installs the chart, creates a demo HTTP Service and `RstreamTunnel`, waits for `Ready`, and curls the forwarding address.

More details are in [docs/RUNTIME_TESTING.md](docs/RUNTIME_TESTING.md).

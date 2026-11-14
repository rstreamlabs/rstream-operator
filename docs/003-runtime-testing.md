# Runtime Testing

This repository has three levels of tests.

## Unit and envtest

```sh
make test
```

This runs:

- generated manifest checks;
- Go formatting and vetting;
- unit tests;
- controller-runtime envtest against a real Kubernetes API server.

## Kind Smoke Test

Use the smoke test when you have a usable project or engine and token:

```sh
export RSTREAM_PROJECT_ENDPOINT="project-endpoint"
export RSTREAM_TOKEN=...
hack/runtime-smoke.sh
```

Set `RSTREAM_API_URL` when testing a staging Control plane. If that Control
plane is protected by one deployment-specific HTTP header, set
`RSTREAM_CONTROL_PLANE_HEADER_NAME` and `RSTREAM_CONTROL_PLANE_HEADER_VALUE`
together; the script stores the value in the Kubernetes credential Secret and
references it through `spec.controlPlaneHeaders`. The header is used only for
Control plane resolution and is never forwarded to the Engine. Set
`RSTREAM_ENGINE` instead of `RSTREAM_PROJECT_ENDPOINT` when the target is
self-hosted or otherwise has no Control plane. `NAMESPACE` may select an
isolated smoke-test namespace; the rendered sample and every generated object
use that exact validated namespace.

The script:

1. creates a Kind cluster if needed;
2. builds the operator image;
3. loads it into Kind;
4. installs the Helm chart;
5. deploys a demo HTTP server;
6. creates `RstreamConnection` and `RstreamTunnel`;
7. waits for `RstreamTunnel/web` to become `Ready`;
8. curls the forwarding address.

The smoke test needs an explicit token because Kubernetes agents cannot read your local CLI context.

For normal hosted installs, prefer `RSTREAM_PROJECT_ENDPOINT` and let the operator resolve the engine through `RSTREAM_API_URL` (`https://rstream.io` by default). For self-hosted or isolated engines, set `RSTREAM_ENGINE` directly and leave `RSTREAM_PROJECT_ENDPOINT` unset.

When a sibling `../rstream-go` checkout exists, `hack/runtime-smoke.sh` builds the image from a temporary context containing both repositories. This keeps unreleased SDK changes testable locally before the public Go module is tagged. Set `RSTREAM_GO_REPO=/path/to/rstream-go` when your local SDK checkout lives somewhere else.

## Local Engine

### Reviewed transport graph

The operator's Go SDK graph uses QUIC v0.64.1, WebTransport v0.13.0, MASQUE
v0.6.0, CONNECT-IP v0.6.0 and DTLS v3.1.15. The standard Go test suite links and
checks this exact tuple with `TestReviewedTransportDependencyGraph`; a source
replacement or drift to another version fails the check.

The qualification branch references SDK source commit `bc2415800ddf` through its
immutable Go pseudo-version. It is not a stable SDK release. Keep consumer review
PRs unmerged until the SDK is certified and its normal stable version is
substituted. Source tests do not publish a package or deploy an operator.

For isolated local smoke tests, use a dedicated `KIND_CLUSTER` and `KUBECONFIG`
file and trust the fixture CA in the test image and HTTP probe. Keep certificate
verification enabled. The smoke scenario also restarts the agent and verifies
that its published tunnel address remains stable.

### Connection settings

For local or self-hosted engine testing, start the engine using that engine repository's instructions, then point the smoke test at it:

```sh
unset RSTREAM_PROJECT_ENDPOINT
export RSTREAM_ENGINE="engine.example.com:443"
export RSTREAM_TOKEN=...
hack/runtime-smoke.sh
```

For local tunnel hostnames, use a resolver path that honors the system resolver. Browsers with Secure DNS enabled may bypass local development DNS resolution.

## Manual Debug Commands

```sh
kubectl -n rstream-demo get rstreamconnection,rstreamtunnel
kubectl -n rstream-demo describe rstreamtunnel web
kubectl -n rstream-demo get deploy -l app.kubernetes.io/name=rstream-operator
kubectl -n rstream-demo logs deploy/$(kubectl -n rstream-demo get rstreamtunnel web -o jsonpath='{.status.agentDeployment}')
```

When the tunnel is ready:

```sh
kubectl -n rstream-demo get rstreamtunnel web -o jsonpath='{.status.forwardingAddress}{"\n"}'
```

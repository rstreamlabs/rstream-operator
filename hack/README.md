# Operator runtime fixture

Use [Contributing](../CONTRIBUTING.md#local-checks) for unit, shell, and
envtest checks. The opt-in [runtime smoke script](runtime-smoke.sh)
exercises a published HTTP request and verifies that the address survives
an agent restart. It needs Docker, Kind, Helm, kubectl, curl, a project or
Engine, and an explicit token.

## Isolate the cluster

Use a cluster name reserved for this test and a dedicated kubeconfig so
the script's Helm and kubectl commands target the same Kind cluster.

```sh
fixture_dir="$(mktemp -d)"
export KUBECONFIG="$fixture_dir/kubeconfig"
export KIND_CLUSTER="rstream-operator-smoke-$(date +%s)"
kind create cluster --name "$KIND_CLUSTER" --kubeconfig "$KUBECONFIG"
```

Set exactly one connection selector and provide the token from a
credential store or restricted local file. The agents authenticate
with the supplied Secret, independently of the local CLI context.
Publishing requires an Engine that serves the returned hostname and a
probe host that resolves and trusts it.

| Environment variable | Purpose and default |
| --- | --- |
| `RSTREAM_PROJECT_ENDPOINT` | Hosted project endpoint. The script uses it first if several selectors are set. |
| `RSTREAM_PROJECT_ID` | Hosted project ID when the endpoint is unavailable. |
| `RSTREAM_ENGINE` | Direct Engine address, usually `host:port`. |
| `RSTREAM_TOKEN` | Required project or Engine token. |
| `RSTREAM_API_URL` | Hosted Control plane URL, defaulting to `https://rstream.io` through the connection. |
| `RSTREAM_CONTROL_PLANE_HEADER_NAME`, `RSTREAM_CONTROL_PLANE_HEADER_VALUE` | Optional pair for hosted lookup through an access gateway. The value goes into a Secret. |
| `KIND_CLUSTER`, `KUBECONFIG` | Kind cluster name and kubeconfig. Set both explicitly for the fixture. |
| `NAMESPACE`, `TIMEOUT` | Sample namespace and wait timeout. Defaults to `rstream-demo` and `180s`. |
| `IMAGE_REPOSITORY`, `IMAGE_TAG` | Local build image, defaulting to `rstream-operator:runtime`. |
| `RSTREAM_GO_REPO` | Optional local SDK checkout. Defaults to the sibling `../rstream-go` when present. |

Set `RSTREAM_PROJECT_ENDPOINT` to an authorized fixture project, or use
`RSTREAM_ENGINE` alone for a direct Engine. Load the token before running
the script.

```sh
export RSTREAM_TOKEN="$(cat /path/to/fixture-token.txt)"
hack/runtime-smoke.sh
unset RSTREAM_TOKEN
```

The script builds and loads the image into Kind, installs the local chart,
deploys the sample, creates both custom resources, waits for readiness,
and requests the forwarding address before and after an agent restart.
When `RSTREAM_GO_REPO` points to an SDK checkout, the script uses both
repositories in a temporary build context. This path also requires
`rsync`.

For a direct Engine with a fixture CA, trust it in the agent image and in
the HTTP probe. The stock script has no option to inject a CA into its
build. Use an Engine certificate trusted by the stock image to run the
script as written. Keep certificate verification enabled. Ensure the
probe's resolver honors any local hostname mappings.

Delete the cluster you created for this fixture.

```sh
kind delete cluster --name "$KIND_CLUSTER" --kubeconfig "$KUBECONFIG"
```

See [Troubleshooting](../docs/005-troubleshooting.md) for status and logs.

## Reviewed SDK transport graph

The operator uses Go SDK v1.38.1. Review the SDK pin and the transport
dependencies together when upgrading.

`TestReviewedTransportDependencyGraph` checks the selected module graph
and linked test binary. The current reviewed versions are QUIC v0.60.0,
WebTransport v0.11.1, MASQUE v0.4.0, CONNECT-IP v0.1.0, and DTLS v3.1.15.

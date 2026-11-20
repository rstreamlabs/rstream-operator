# Installation reference

The [installation guide](000-getting-started.md) pins the chart and image to a
published release. To render or install from a checkout, use
`./charts/rstream-operator` as the chart argument, omit the OCI `--version`
flag, and set `image.repository` and `image.tag` to the intended image.

## Helm values

| Value | Chart default | Effect |
| --- | --- | --- |
| `image.repository`, `image.tag` | `rstream/rstream-operator`, `latest` | Manager and default agent image. Use the reviewed release tag for a reproducible install. |
| `image.pullPolicy` | `IfNotPresent` | Manager image pull policy. Managed agents use `IfNotPresent`. |
| `agent.image.repository`, `agent.image.tag` | Empty, empty | Set the agent repository to select a separate image. Its tag falls back to the manager tag when omitted. `RstreamTunnel.spec.agent.image` takes precedence for an individual tunnel. |
| `imagePullSecrets` | Empty list | Registry credentials for the manager Pods. Managed agents get their own ServiceAccount in the tunnel namespace. |
| `replicaCount`, `leaderElection.enabled` | `1`, `true` | Manager replicas and lease-based leadership. Keep leader election enabled when running multiple manager replicas. |
| `metrics.bindAddress` | `:8080` | Manager metrics listener. Use `"0"` to disable it. |
| `health.bindAddress` | `:8081` | Manager health and readiness listener. Keep port 8081 with the stock chart probes. |
| `resources` | Requests 50m CPU and 128Mi memory, limit 512Mi memory | Manager Pod resource settings. Set `spec.agent.resources` on each tunnel to change its agent resources. |
| `nodeSelector`, `tolerations`, `affinity` | Empty | Manager scheduling settings. |
| `podAnnotations`, `podLabels` | Empty | Manager Pod metadata. |
| `securityContext`, `containerSecurityContext` | Non-root, RuntimeDefault seccomp, read-only container filesystem, no added privileges | Manager Pod and container security settings. |
| `nameOverride`, `fullnameOverride` | Empty | Overrides used to name and label chart resources. |

The full [values file](../charts/rstream-operator/values.yaml) is packaged with
the chart. Helm installs the CRDs from `crds/`. If CRDs are managed separately,
use Helm's `--skip-crds` on installation. The chart's `installCRDs` value has
no effect.

## Manager and agent options

Helm sets manager flags. Use these options when running a binary directly or
reading the arguments on an installed Pod.

| Process | Option | Binary default |
| --- | --- | --- |
| Manager | `--agent-image` | `RSTREAM_AGENT_IMAGE` when set, otherwise `ghcr.io/rstreamlabs/rstream-operator:latest`. Helm passes the configured agent image explicitly. Use the published Docker Hub image for a standalone release install. |
| Manager | `--leader-elect` | `false`. Helm enables it by default. |
| Manager | `--metrics-bind-address`, `--health-probe-bind-address` | `:8080`, `:8081` |
| Manager | `--zap-devel`, `--zap-encoder`, `--zap-log-level`, `--zap-stacktrace-level`, `--zap-time-encoding` | Production logging uses JSON, info level, error stack traces, and epoch timestamps. |
| Manager, agent | `--kubeconfig` | Empty. Use a kubeconfig when running outside Kubernetes. |
| Agent | `--config`, `--health-bind-address` | `/etc/rstream-agent/config.yaml`, `:8081` |

| Environment variable | Scope |
| --- | --- |
| `KUBECONFIG` | Selects the kubeconfig file for Helm and kubectl. Set it explicitly when using a dedicated local cluster. |
| `RSTREAM_AGENT_IMAGE` | Standalone manager fallback for the default agent image. `--agent-image` takes precedence. |
| `RSTREAM_AUTHENTICATION_TOKEN` | Injected into token-authenticated agent Pods from `spec.tokenSecretRef`. |
| `RSTREAM_PROXY_USERNAME`, `RSTREAM_PROXY_PASSWORD` | Injected into agent Pods when the corresponding proxy Secret references are configured. |
| `RSTREAM_AGENT_POD_NAME`, `RSTREAM_AGENT_POD_NAMESPACE` | Injected from the agent Pod metadata. |

Set credentials through `RstreamConnection` Secret references when using the
Operator. See [Architecture and credentials](002-architecture.md).

## Upgrade or remove the chart

Review the [release notes](https://github.com/rstreamlabs/rstream-operator/releases),
including CRD changes. Helm leaves existing CRDs in place during upgrade.
Set `OPERATOR_VERSION` to the target release. Apply that release's CRDs
before upgrading the manager if it includes schema changes.

```sh
helm show crds oci://ghcr.io/rstreamlabs/rstream-operator --version "$OPERATOR_VERSION" \
  > rstream-operator-crds.yaml
kubectl apply --server-side -f rstream-operator-crds.yaml
```

Run the [install command](000-getting-started.md) with the target version.
Delete the tunnels while the manager is still running before uninstalling
the release.

```sh
helm uninstall rstream-operator --namespace rstream-system
```

Helm preserves the CRDs and existing custom resources during uninstall. Removing
the CRDs separately deletes their corresponding custom resources across the
cluster.

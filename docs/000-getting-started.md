# Install the Operator

Install the published Helm chart to add the `tunnels.rstream.io/v1alpha1` API and
run the controller. You can verify the installation before configuring any
rstream credentials.

| Prerequisite | What you need |
| --- | --- |
| Kubernetes | A cluster running Kubernetes 1.29 or newer. |
| kubectl and Helm | Both clients configured for the same cluster. |
| Cluster permissions | Permission to install CustomResourceDefinitions, cluster RBAC, and the manager Deployment. |
| Registry access | The Helm client needs access to `ghcr.io`. Kubernetes nodes need access to Docker Hub for the release image. |

The chart installs a cluster-wide manager role. Its ServiceAccount can read
and watch Secrets across managed namespaces. Review that access before
installation; [the Operator architecture](002-architecture.md) describes the
boundary between the manager and each tenant’s tunnel agent.

## Install a stable release

Check that kubectl is targeting the cluster where you want to install the Operator.

```sh
kubectl config current-context
```

Install the chart and matching image from the [stable release](https://github.com/rstreamlabs/rstream-operator/releases).

```sh
OPERATOR_VERSION=0.6.2
helm upgrade --install rstream-operator oci://ghcr.io/rstreamlabs/rstream-operator \
  --version "$OPERATOR_VERSION" \
  --namespace rstream-system \
  --create-namespace \
  --set-string image.tag="$OPERATOR_VERSION" \
  --wait --timeout 3m
```

Helm pulls the chart from GitHub Container Registry. The manager and its
agents use `docker.io/rstream/rstream-operator:0.6.2` with these settings. The
chart's `--version` selects only the chart, so set the image tag explicitly.

## Verify the installation

```sh
kubectl wait --for=condition=Established \
  crd/rstreamconnections.tunnels.rstream.io \
  crd/rstreamtunnels.tunnels.rstream.io --timeout=60s
kubectl -n rstream-system rollout status deployment/rstream-operator --timeout=180s
kubectl api-resources --api-group=tunnels.rstream.io
```

The manager Deployment should finish its rollout and the API resource list
should include namespaced `rstreamconnections` and `rstreamtunnels`.

Continue with [Publish your first Service](001-first-tunnel.md). For chart
overrides and release upgrades, see the [installation reference](004-installation-reference.md).

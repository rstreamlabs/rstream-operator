# Install and create a tunnel

Use Kubernetes 1.29 or newer, kubectl, Helm and a rstream project-scoped token. Run these commands from a checkout of this repository. For compilation without a cluster or credentials, use [Build from source](001-building-from-source.md).

The chart defaults to the `latest` operator image and lets the agent inherit it. For a reproducible installation, select a reviewed image tag with `--set image.tag=<reviewed-tag>` in the Helm command below. The chart version and `appVersion` are metadata, not an override of `values.yaml`.

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

Set `RSTREAM_TOKEN` to a token authorized for this project, then create the credential Secret:

```sh
kubectl -n rstream-demo create secret generic rstream-credentials \
  --from-literal=token="$RSTREAM_TOKEN"
```

Save the connection and tunnel below as `rstream-tunnel.yaml`:

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: rstream-demo
spec:
  projectEndpoint: "my-project-endpoint"
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

Apply the file, then check readiness and the public address:

```sh
kubectl apply -f rstream-tunnel.yaml
kubectl -n rstream-demo wait --for=condition=Ready rstreamtunnel/web --timeout=120s
kubectl -n rstream-demo get rstreamtunnel web
kubectl -n rstream-demo describe rstreamtunnel web
```

Replace `my-project-endpoint` with the project endpoint from the dashboard. The sample Service creates the `rstream-demo` namespace used by the credentials and custom resources. For an existing Service, use its namespace, name and port instead.

The forwarding address appears when `Ready` is true. Copy `.status.forwardingAddress` from the resource and request it with `curl`; the sample Python HTTP server returns an HTML directory listing. A successful request checks the published data path as well as reconciliation readiness. Readiness alone does not prove a particular application response. See [Troubleshooting](005-troubleshooting.md) if resolution or agent readiness fails. [API reference](003-api-reference.md) covers self-hosted engines, certificate credentials and additional protocol settings.

# Install and create a tunnel

Use a Kubernetes cluster, kubectl, Helm and a rstream project-scoped token. Run these commands from a checkout of this repository.

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

Create the rstream connection and tunnel:

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

Check readiness and the public address:

```sh
kubectl -n rstream-demo get rstreamtunnel web
kubectl -n rstream-demo describe rstreamtunnel web
```

Replace `my-project-endpoint` with the project endpoint from the dashboard. The sample Service creates the `rstream-demo` namespace used by the credentials and custom resources. For an existing Service, use its namespace, name and port instead.

The forwarding address appears when `Ready` is true. See [Troubleshooting](004-troubleshooting.md) if resolution or agent readiness fails. [API reference](002-api-reference.md) covers self-hosted engines, certificate credentials and additional protocol settings.

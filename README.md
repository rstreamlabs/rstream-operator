# rstream Kubernetes Operator

The rstream Operator can give a `ClusterIP` Service an HTTPS address.
The Service stays internal to Kubernetes. An agent Pod opens an outbound
connection from the cluster and forwards incoming requests to the Service.
For access limited to authenticated rstream clients, the Operator can also
create a private tunnel.

## Publish an HTTP Service

On a new cluster, start with [the Helm installation](docs/000-getting-started.md)
and continue to the sample HTTP Service. For an existing Service, use a connection in its namespace or
[create one](docs/001-first-tunnel.md#2-configure-the-connection). The
`RstreamConnection` selects a hosted project or a
self-hosted Engine and references credentials in a Kubernetes Secret.

An `RstreamTunnel` declares which Service port to expose. The manifest below
publishes the `http` port of `http-server` using the `default` connection in
`rstream-demo`. The project or Engine must allow HTTP publication. To use
another Service, substitute its namespace, name, port and connection name.
Save the manifest as `tunnel.yaml`.

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
  namespace: rstream-demo
spec:
  connectionRef:
    name: default
  target:
    service:
      name: http-server
      port: http
  publish: true
  protocol: http
  http:
    version: http/1.1
```

Apply the manifest. The controller creates a dedicated agent Pod for the tunnel.

```sh
kubectl apply -f tunnel.yaml
```

Check the tunnel status and forwarding address.

```sh
kubectl get -f tunnel.yaml
```

Once `READY` is `True`, open the URL in `ADDRESS` to reach the Service.
The sample HTTP server returns `hello from rstream Operator`. To withdraw
the endpoint, delete the tunnel. The Service remains available inside the
cluster.

```sh
kubectl delete -f tunnel.yaml
```

## Operations and development

The [Operator documentation](docs/README.md) covers Helm values, credentials,
API fields and troubleshooting. See [Contributing](CONTRIBUTING.md) to build
or test the controller and agent.

Report reproducible bugs in [GitHub issues](https://github.com/rstreamlabs/rstream-operator/issues).
Use [Security](SECURITY.md) for private vulnerability reports. Licensed under
[Apache-2.0](LICENSE).

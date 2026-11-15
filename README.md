# rstream Kubernetes operator

rstream makes private services reachable through outbound connections instead of requiring inbound access to their hosts. This operator applies that model to Kubernetes Services using namespaced custom resources.

Declare a connection and a tunnel alongside the application resources. The controller reconciles their configuration and lifecycle; a dedicated agent Pod for each tunnel carries the traffic to the target Service. The application does not need to embed an SDK.

| Resource            | Responsibility                                                                         |
| ------------------- | -------------------------------------------------------------------------------------- |
| `RstreamConnection` | Shared project or engine settings and references to credentials in Kubernetes Secrets. |
| `RstreamTunnel`     | A tunnel to a Service, with its protocol, publication policy and readiness status.     |

The controller reconciles Kubernetes resources; the Go-based agent establishes outbound connectivity to the edge and forwards traffic to the target Service. The API is `tunnels.rstream.io/v1alpha1`.

## Create a tunnel

Follow [Install and create a tunnel](docs/000-getting-started.md) to install the Helm chart, configure a project and create a sample Service. Once a `default` connection exists, a tunnel to that Service looks like this:

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

`kubectl -n rstream-demo get rstreamtunnel web` shows readiness and the forwarding address. Private tunnels use the same resource model with publication disabled. The [API reference](docs/003-api-reference.md) describes the supported settings.

## Documentation

| Task                                                | Guide                                                                              |
| --------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Install the operator and forward a Service          | [Getting started](docs/000-getting-started.md)                                     |
| Compile the controller and agent | [Build from source](docs/001-building-from-source.md) |
| Configure connections, credentials or tunnel policy | [API reference](docs/003-api-reference.md)                                         |
| Understand controllers, agents and ownership        | [Architecture](docs/002-architecture.md)                                           |
| Diagnose readiness and reconnect failures           | [Troubleshooting](docs/005-troubleshooting.md)                                     |
| Develop or test the operator                        | [Contributing](CONTRIBUTING.md) and [runtime testing](docs/004-runtime-testing.md) |

Release images are distributed as `rstream/rstream-operator`. For application-level integration outside Kubernetes, use the [Go reference SDK](https://github.com/rstreamlabs/rstream-go) or another [rstream SDK](https://rstream.io/sdks).

## Security and support

Credentials remain in Secrets and are not copied into agent ConfigMaps. Agent Pods run without root privileges and use per-tunnel RBAC. See [Architecture](docs/002-architecture.md) for the ownership model and [Security](SECURITY.md) for private vulnerability reporting. Usage questions and reproducible bugs belong in [GitHub issues](https://github.com/rstreamlabs/rstream-operator/issues), with secrets and customer details removed. Licensed under [Apache-2.0](LICENSE).

# rstream Kubernetes Operator

The rstream Operator forwards traffic to Kubernetes Services over outbound
connections to rstream. Install the chart, create a connection with a
Kubernetes Secret, then declare which Service and port to forward. Your
application keeps serving traffic through its existing Service.

| Resource | Responsibility |
| --- | --- |
| `RstreamConnection` | Specifies a hosted rstream project or a direct Engine address and references credentials. |
| `RstreamTunnel` | Selects a Service port and defines the tunnel protocol and publication policy. |

The controller reconciles these resources. One managed agent Pod per tunnel
opens the outbound connection and forwards traffic to the Service.

Start with [Install the Operator](docs/000-getting-started.md), then
[publish a sample HTTP Service](docs/001-first-tunnel.md). The
[documentation index](docs/README.md) covers architecture, API fields, Helm
settings, and troubleshooting. See [Contributing](CONTRIBUTING.md) to build or
test the controller and agent.

Report reproducible bugs in [GitHub issues](https://github.com/rstreamlabs/rstream-operator/issues).
Use [Security](SECURITY.md) for private vulnerability reports. Licensed under
[Apache-2.0](LICENSE).

# Contributing to the Kubernetes operator

The manager owns reconciliation and Kubernetes resource state; agents own tunnel connections and forwarding. Keep changes within those boundaries and update the [API reference](docs/003-api-reference.md) when changing a custom resource.

## Local checks

| Tool | Requirement |
| --- | --- |
| Go | The version in `go.mod` or newer; this checkout declares Go 1.27.4. |
| Git, Make and a POSIX shell | Source checkout and the repository build/tool targets. |
| C compiler | Required for supported race-detector runs. |
| kubectl and Helm | Install/render the operator and test the documented Kubernetes resources. |
| Docker and Kind | Only for the optional local-cluster runtime fixture and image workflows. |

The Makefile installs its versioned development tools under `bin/`, including envtest assets for local controller tests. A running Kubernetes cluster is not needed to compile the manager. From the repository root:

```sh
make generate
make manifests
make test
make helm-lint
make build
```

`make build` writes the controller to `bin/manager` and the forwarding agent to `bin/rstream-agent`. `make test` starts isolated envtest control-plane processes; application-to-engine runtime coverage follows the separate fixture below. Inspect generated CRDs and their Helm copies together. Tests should cover rejected specs, credential isolation and status ownership as well as successful reconciliation.

For SDK development, a local Go workspace can include a reviewed SDK checkout. Keep `go.work` uncommitted; `go.mod` remains a normal immutable module dependency for release builds. The [runtime testing guide](docs/004-runtime-testing.md) describes the opt-in Kind/engine fixture and explicit credential prerequisites.

Use [Security](SECURITY.md) for vulnerability reports. Image and Helm publication are separate maintainer operations.

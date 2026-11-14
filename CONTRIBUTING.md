# Contributing to the Kubernetes operator

The manager owns reconciliation and Kubernetes resource state; agents own tunnel connections and forwarding. Keep changes within those boundaries and update the [API reference](docs/002-api-reference.md) when changing a custom resource.

## Local checks

The Makefile keeps tools under `bin/`. From the repository root:

```sh
make generate
make manifests
make test
make helm-lint
make build
```

Inspect generated CRDs and their Helm copies together. Tests should cover rejected specs, credential isolation and status ownership as well as successful reconciliation.

For SDK development, a local Go workspace can include a reviewed SDK checkout. Keep `go.work` uncommitted; `go.mod` remains a normal immutable module dependency for release builds. The [runtime testing guide](docs/003-runtime-testing.md) describes the opt-in Kind/engine fixture and explicit credential prerequisites.

Use [Security](SECURITY.md) for vulnerability reports. Image and Helm publication are separate maintainer operations.

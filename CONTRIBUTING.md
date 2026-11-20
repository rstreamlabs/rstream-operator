# Contributing to the Kubernetes Operator

The manager owns reconciliation and Kubernetes resource state. Agents own
tunnel connections and forwarding. Keep changes within those boundaries
and update the [API reference](docs/003-api-reference.md) when changing a
custom resource.

## Build from source

Use the Go version declared in `go.mod` and make sure Go can download the
locked modules. Build both binaries from the repository root.

```sh
go version
make build
./bin/manager --help
./bin/rstream-agent --help
```

The binaries are `bin/manager` and `bin/rstream-agent`. Compilation and
`--help` run without a Kubernetes cluster or rstream credentials. For a
chart installation, follow [Install the Operator](docs/000-getting-started.md).

## Local checks

| Tool | Requirement |
| --- | --- |
| Git, Make, and a POSIX shell | Source checkout and the repository build and tool targets. |
| C compiler | Supported race-detector runs. |
| Helm | Chart rendering and lint. |
| kubectl, Docker, and Kind | The opt-in runtime fixture. |

```sh
make test
make helm-lint
```

`make test` installs versioned tools and envtest assets under `bin/`.
It synchronizes the Helm CRD copies, runs formatting, vetting, and shell
tests, and starts local envtest processes for controller tests.
Run `make generate helm-crds` to regenerate API and chart files on their
own. Review generated CRDs with their Helm copies. Tests should cover
rejected specs, credential isolation and status ownership as well as
reconciliation.

A local Go workspace can include a reviewed SDK checkout. Keep `go.work`
uncommitted so release builds continue using the immutable module
dependency in `go.mod`. See [the runtime fixture](hack/README.md) for the
Kind and Engine test, including its explicit credential prerequisites.

Use [Security](SECURITY.md) for vulnerability reports.

# Build the controller and agent

The controller reconciles Kubernetes resources; the agent carries tunnel traffic.
Build both when changing their shared configuration. This check needs no project,
cluster or token and does not install either process into Kubernetes.

Use the Go version in `go.mod` (Go 1.27.4 in this checkout), Git and a shell.
Locked modules must be available locally or downloadable. From the repository root:

```sh
go version
go build -o bin/manager ./cmd/manager
go build -o bin/rstream-agent ./cmd/rstream-agent
./bin/manager --help
./bin/rstream-agent --help
```

The binaries report their options and exit. Running them requires Kubernetes
configuration and the appropriate connection or agent inputs. For an operational
first tunnel, continue with [Getting started](000-getting-started.md).

## Develop and validate

The Makefile also needs Make and POSIX utilities; race tests need a supported C
compiler. Its generated-code tools are pinned in the Makefile:

```sh
make generate manifests
make test
make helm-lint
make build
```

`make test` prepares envtest assets and runs a local Kubernetes API server, not
an existing cluster. Asset preparation may download the selected Kubernetes
tools. It also synchronizes Helm CRD copies and runs formatting, vetting and script
tests. Review generated changes together with API changes. Helm is required for
chart checks. See [Contributing](../CONTRIBUTING.md) for ownership and prerequisites.

Docker and Kind are needed only for the separate [runtime smoke test](004-runtime-testing.md).
That path creates or changes cluster resources and needs explicit project or
engine credentials. Compilation and envtest do not establish runtime connectivity.

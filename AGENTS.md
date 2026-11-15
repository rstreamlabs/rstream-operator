# Repository guidance

## Shared conventions

Follow the existing architecture, naming and coding style. Preserve public contracts, authentication, certificate verification and project policy. Keep changes in the component that owns the behavior.

Update user documentation and examples when behavior changes. Run the relevant checks; place regression tests with the owning component. Keep secrets, execution logs, investigation notes and standalone upstream reproductions outside the repository.

## Repository-specific guidance

Follow `CONTRIBUTING.md`. The controller owns Kubernetes reconciliation; agent Pods own runtime traffic. Keep generated CRDs and Helm copies aligned. Preserve credential isolation and status ownership. Use isolated envtest/Kind fixtures and explicit credentials for runtime tests.

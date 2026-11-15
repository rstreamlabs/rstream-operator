# Kubernetes operator documentation

Start with [Install and create a tunnel](000-getting-started.md) for a running
cluster, or [Build from source](001-building-from-source.md) for a credential-free
local check. The controller manages agents for Services; application traffic
flows through the agents.

| Guide | Purpose |
| --- | --- |
| [000 · Getting started](000-getting-started.md) | Install the chart, configure credentials and request a Service tunnel. |
| [001 · Build from source](001-building-from-source.md) | Compile both processes and choose local checks. |
| [002 · Architecture](002-architecture.md) | Understand reconciliation, ownership and failures. |
| [003 · API reference](003-api-reference.md) | Set connections, targets, transport and policy. |
| [004 · Runtime testing](004-runtime-testing.md) | Qualify the controller and agent against an explicit fixture. |
| [005 · Troubleshooting](005-troubleshooting.md) | Interpret readiness and diagnose connection problems. |

[Contributing](../CONTRIBUTING.md) describes API generation and chart maintenance.

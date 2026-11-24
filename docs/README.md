# SENTINEL documentation

Start with the root [README](../README.md) for the short project pitch and
quickstart. These documents describe the implementation and operating model:

| Document | What it covers |
|---|---|
| [Architecture](architecture.md) | Components, request pipeline, state ownership, and failure behavior |
| [API reference](api.md) | Gateway and operational endpoints, headers, authentication, and responses |
| [Configuration](configuration.md) | YAML settings, environment overrides, and defaults |
| [Operations runbook](operations.md) | Local/Compose operation, health checks, incident drills, and shutdown |
| [Observability](observability.md) | Metrics, labels, PromQL examples, logs, and dashboards |
| [Security](security.md) and [reporting policy](../SECURITY.md) | Threat model, controls, trust boundaries, and vulnerability disclosure |
| [Development](development.md) | Tooling, code map, test commands, and contribution workflow |
| [Performance testing](performance.md) | k6 scenario, measurement method, and benchmark reporting |
| [Release and CI/CD](release.md) | GitHub Actions validation, GHCR publishing, and deployment boundary |
| [Interview notes](interview-notes.md) | Senior-level discussion prompts and concise design answers |
| [Architecture decisions](adr/README.md) | Decision records and the reasoning behind key tradeoffs |

Documentation is kept evidence-based: benchmark results are intentionally
blank until measured, and deployment instructions do not imply that an image
has been deployed to a production environment.

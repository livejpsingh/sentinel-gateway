# Development guide

## Prerequisites

- Go 1.22 or newer
- Git
- Docker Desktop with Compose plugin for the full local stack
- Optional: k6 for load testing

## Repository map

```text
cmd/sentinel/          Application entrypoint and server lifecycle
internal/config/       YAML parsing, env overrides, validation
internal/proxy/        Reverse proxy and per-upstream breaker registry
internal/middleware/   HTTP middleware and focused tests
internal/metrics/      Prometheus collectors and registry
internal/health/       Health and readiness handlers
mocks/users/           users-service development mock
mocks/payments/        payments-service development mock
config/                Local gateway configuration
deploy/                Dockerfile and Docker Compose stack
prometheus/            Prometheus scrape configuration
loadtest/              k6 ramping scenario
docs/                  Project and operating documentation
```

## Build, test, and vet

From the repository root:

```powershell
go mod download
go test -race ./...
go vet ./...
go build ./cmd/sentinel ./mocks/users ./mocks/payments
```

Compose can also build/run all services:

```powershell
docker compose -f .\deploy\docker-compose.yml up --build
```

## Local processes

Open three terminals and run:

```powershell
go run .\mocks\users
go run .\mocks\payments
go run .\cmd\sentinel
```

For local development, create a short-lived HS256 token using the PowerShell
snippet in the root [README](../README.md), then send it as a Bearer token.
Never use the documented demo secret outside a local demo.

## Test coverage

Focused table-driven tests cover rate-bucket isolation/rejection and JWT
validation cases. Configuration tests verify YAML duration parsing and
environment overrides. Run `-race` because the limiter, metrics, and circuit
breakers are used concurrently.

When adding behavior, add tests at the existing package boundary. For middleware
tests, use `httptest.NewRequest` and `httptest.NewRecorder`; do not bind network
ports unless integration behavior requires it.

## Change workflow

1. Keep changes narrowly scoped and update relevant docs.
2. Add/adjust focused tests for behavior changed.
3. Run race tests, vet, and builds.
4. Validate the Compose file with
   `docker compose -f .\deploy\docker-compose.yml config`.
5. Avoid adding raw path, token, request ID, or user identifiers as metric
   labels.
6. Do not commit credentials, local environment files, generated binaries, or
   benchmark results that were not measured.

Pull requests run the same Go validation in GitHub Actions. See
[release and CI/CD](release.md).

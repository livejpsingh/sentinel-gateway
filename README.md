# SENTINEL

> **A Go API gateway capstone for backend interviews:** a standard-library
> reverse proxy with JWT authentication, per-client rate limiting, independent
> upstream circuit breakers, bounded request sizes, graceful shutdown, and
> Prometheus metrics. Its design tradeoffs and operational limits are documented
> so the project can be demonstrated and defended rather than treated as a
> black-box demo.

## Architecture

```text
Client
  → Request ID → JSON logger → Recovery
  → API: Metrics → Route timeout → Circuit breaker → JWT → Rate limit
      → Body limit → ReverseProxy → users-service | payments-service
  → Operational routes: /healthz | /readyz | /metrics
```

The gateway uses plain `net/http` and `httputil.ReverseProxy`. Rate and breaker
state are per process; upstream connections have configured caps and requests
have route deadlines. Metrics use fixed route labels. `/metrics` and health
probes are unauthenticated and should be restricted at the network boundary in
non-local environments.

See the [documentation index](docs/README.md) for the architecture, API,
configuration, operations, security, observability, development, release,
performance, interview, and architecture-decision docs.

## Benchmarks

No performance claims are published without measurements on a stated host and
configuration. Fill this table using the procedure in
[performance testing](docs/performance.md).

| Metric | Value |
|---|---|
| RPS | ___ |
| P99 latency | ___ |
| Memory (RSS) | ___ |
| Goroutines at peak | ___ |

The supplied [k6 scenario](loadtest/k6.js) ramps to 5,000 VUs. This is a test
target, not a throughput guarantee.

## How to run

Requirements: Go 1.22+ for local execution, or Docker Compose for the complete
stack. The default JWT key is for local demonstration only.

### Docker Compose

From the repository root:

```powershell
docker compose -f .\deploy\docker-compose.yml up --build
```

The gateway listens on `http://localhost:8080`; Prometheus is available at
`http://localhost:9090`; mock upstreams are internal to the Compose network.
`/metrics`, `/healthz`, and `/readyz` can be called without a token. If port
9090 is occupied, set `$env:PROMETHEUS_PORT = "9091"` before starting Compose.

Generate a short-lived development JWT in PowerShell, using the Compose
development key:

```powershell
$secret = "compose-development-secret-change-this-before-production"
$b64 = { param($value) [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($value)).TrimEnd('=').Replace('+','-').Replace('/','_') }
$header = & $b64 '{"alg":"HS256","typ":"JWT"}'
$payload = & $b64 ('{"sub":"demo-user","iss":"sentinel-local","aud":"sentinel-api","exp":' + [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() + '}')
$signingInput = "$header.$payload"
$hmac = [System.Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
$signature = [Convert]::ToBase64String($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($signingInput))).TrimEnd('=').Replace('+','-').Replace('/','_')
$env:JWT_TOKEN = "$signingInput.$signature"
curl.exe -i -H "Authorization: Bearer $env:JWT_TOKEN" http://localhost:8080/api/users/42
```

Stop the stack with `Ctrl+C`, or run
`docker compose -f .\deploy\docker-compose.yml down`.

### Local Go processes

Run each command in its own terminal from the repository root:

```powershell
go run .\mocks\users
go run .\mocks\payments
go run .\cmd\sentinel
```

For local mode, sign the same token using the `auth.secret` value in
`config/config.yaml`. See [API usage](docs/api.md) and
[development](docs/development.md).

## CI/CD

GitHub Actions runs race-enabled tests, vet, and builds on pull requests and
pushes. Validated pushes to `main` and version tags publish the gateway image
to GHCR. This is continuous delivery, not deployment to an unspecified
production environment. See [release and CI/CD](docs/release.md).

## Reliability drills

The [operations runbook](docs/operations.md) gives exact steps to demonstrate
upstream failure and circuit breaking, rate-limit rejection, and graceful
shutdown. Expected status codes and metrics are in the [API reference](docs/api.md)
and [observability guide](docs/observability.md).

## Scope and limitations

This capstone intentionally does not implement distributed rate limits or
breakers, tenant/role authorization, JWT key rotation/JWKS, TLS termination,
service discovery, dynamic config reload, or OpenTelemetry spans/exporters.
See [security notes](docs/security.md) and the [architecture decisions](docs/adr/README.md)
for the consequences and production follow-up items.

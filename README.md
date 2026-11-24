# SENTINEL

> **A production-minded Go API gateway capstone:** a small, dependency-light
> reverse proxy with JWT authentication, per-client rate limiting, independent
> upstream circuit breakers, bounded request sizes, graceful shutdown, and
> Prometheus metrics. It is designed to make the reliability and security
> decisions visible and explainable in a senior backend interview.

## Architecture

```text
Client
  │
  ▼
Request ID → JSON request log → Panic recovery → Body limit
  → Per-client token bucket → HS256 JWT validation
  → Per-upstream circuit breaker → Route deadline → Prometheus timing
  → net/http/httputil ReverseProxy
      ├── /api/users/*    → users-service
      └── /api/payments/* → payments-service

/healthz and /readyz → operational probes
/metrics             → private Prometheus registry
```

The gateway uses ordinary `net/http` handlers and middleware rather than a web
framework. `ReverseProxy` owns request forwarding and hop-by-hop header removal;
the route handler strips the public API prefix before forwarding. X-Request-ID
is preserved when supplied (up to 128 characters) and otherwise generated as a
UUIDv4. W3C `traceparent`/`tracestate` headers are forwarded as a tracing-ready
stub; no exporter or span SDK is configured.

### Design tradeoffs

- **In-memory token buckets:** `golang.org/x/time/rate` provides a predictable
  token bucket with low per-request overhead. State is bounded and idle entries
  are removed. Limits are per gateway process and reset on restart; use a
  distributed limiter (for example Redis) when replicas must enforce one shared
  quota. This sample keys by socket client IP and deliberately does not trust
  caller-supplied forwarding headers.
- **ReverseProxy over a hand-built proxy:** the standard library handles HTTP
  forwarding, request cancellation, upgrades, and RFC hop-by-hop headers.
- **Circuit breaker:** each upstream has closed/open/half-open states. Five
  consecutive failed responses open the breaker; it stays open for ten seconds
  and then permits one probe. The single half-open request limits recovery
  herds. The breaker is per process, not a distributed coordination mechanism.
- **Bounded metrics cardinality:** routes are fixed labels, not raw request
  paths or arbitrary client identifiers. This keeps Prometheus series bounded.
- **Backpressure without a worker pool:** each upstream transport caps active
  connections with `MaxConnsPerHost` (1,024 by default). Excess calls wait in
  the standard transport and the per-route deadline bounds that wait. This
  avoids a second goroutine/queue layer; tune the connection cap from measured
  upstream capacity and latency.
- **Readiness semantics:** `/readyz` reflects whether this gateway is accepting
  work; an upstream outage does not make the gateway itself unready. Breaker
  state and upstream errors are available in metrics.

### Failure modes

| Condition | Client behavior | Gateway behavior |
|---|---|---|
| Upstream is down | 502/504 while failures accumulate, then fast 503 while breaker is open | Counts failed responses; separate breaker state per upstream |
| Upstream exceeds route deadline | 504 | Cancels the request context; records the failed response in the breaker |
| Upstream returns 5xx | Original upstream response is passed through | Counts consecutive failure; opens breaker at threshold |
| Rate budget exhausted | 429 with `Retry-After: 1` | Rejects before JWT parsing or upstream work |
| Body exceeds configured maximum | 413 | Rejects known oversized `Content-Length` immediately; streaming bodies are capped by `MaxBytesReader` |
| Shutdown begins | New connections stop; active handlers drain | Waits up to the configured shutdown deadline (15s by default) |

### Metrics

`/metrics` exposes a dedicated Prometheus registry:

- `sentinel_http_requests_total{method,route,status}`
- `sentinel_http_request_duration_seconds{method,route}`
- `sentinel_rate_limit_rejected_total{route}`
- `sentinel_circuit_breaker_state{upstream}` (`0` closed, `1` half-open, `2` open)
- `sentinel_upstream_errors_total{upstream}`

Do not expose `/metrics` to the public internet in a production deployment;
restrict it at the network/ingress layer.

## Benchmarks

No performance claims are published without measurements on a stated host and
configuration. Fill this table with results from your target environment.

| Metric | Value |
|---|---|
| RPS | ___ |
| P99 latency | ___ |
| Memory (RSS) | ___ |
| Goroutines at peak | ___ |

Use the provided k6 ramping scenario at `loadtest/k6.js`. Its 5,000 VUs are a
test target, not a claim that one laptop or container will sustain that load.
Benchmark with realistic upstream behavior, authentication, machine sizing,
and network conditions; the configured in-memory limiter defaults to 50,000
requests/second per client IP with a 100,000-request burst.

## How to run

Requirements: Go 1.22+ for local execution, or Docker Compose for the container
stack. The supplied config's HS256 secret is for local development only.

## CI/CD

The GitHub Actions workflow at `.github/workflows/ci-cd.yml` runs race-enabled
tests, `go vet`, and builds all three Go binaries on pull requests and pushes
to `main` or version tags. It validates against Go 1.22 and the current stable
release. After validation, pushes to `main` and tags matching `v*` publish the
gateway image to `ghcr.io/livejpsingh/sentinel-gateway` with branch/tag, commit,
and (on the default branch) `latest` tags. The workflow uses the repository's
`GITHUB_TOKEN`; grant the Actions workflow permission to write packages if
repository settings require it.

This pipeline provides continuous delivery of a tested image, not deployment
to a production host: no deployment environment or credentials were supplied.
Deploy the published image through your environment's reviewed release process,
or extend the workflow with environment-protected deployment jobs once a
target platform and rollback policy are chosen.

### Local Go processes

Run each command in its own terminal from the repository root:

```powershell
go run .\mocks\users
go run .\mocks\payments
go run .\cmd\sentinel
```

Generate a short-lived development JWT with PowerShell/.NET and call the API.
Set `$secret` to the same value as `auth.secret` in your local config:

```powershell
$secret = "local-development-secret-change-this-before-production"
$b64 = { param($value) [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($value)).TrimEnd('=').Replace('+','-').Replace('/','_') }
$header = & $b64 '{"alg":"HS256","typ":"JWT"}'
$payload = & $b64 ('{"sub":"demo-user","iss":"sentinel-local","aud":"sentinel-api","exp":' + [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() + '}')
$signingInput = "$header.$payload"
$hmac = [System.Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
$signature = [Convert]::ToBase64String($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($signingInput))).TrimEnd('=').Replace('+','-').Replace('/','_')
$env:JWT_TOKEN = "$signingInput.$signature"
curl.exe -H "Authorization: Bearer $env:JWT_TOKEN" http://localhost:8080/api/users/42
curl.exe http://localhost:8080/healthz
curl.exe http://localhost:8080/readyz
curl.exe http://localhost:8080/metrics
```

The JWT secret, issuer, and audience can be overridden with
`SENTINEL_JWT_SECRET`, `SENTINEL_JWT_ISSUER`, and
`SENTINEL_JWT_AUDIENCE`. Do not reuse the development key.

### Docker Compose

From the repository root:

```powershell
docker compose -f .\deploy\docker-compose.yml up --build
```

The gateway listens on `http://localhost:8080`, Prometheus on `http://localhost:9090`, and mock services remain on the internal
Compose network. `/metrics` is immediately available without an API token. Use
the same JWT-generation snippet above, setting `$secret` to the Compose secret:
`compose-development-secret-change-this-before-production`. If host port 9090
is occupied, set `$env:PROMETHEUS_PORT = "9091"` before starting Compose; the
Prometheus container still listens on 9090 internally.

### Configuration and environment

The YAML file is `config/config.yaml` (override its path with
`SENTINEL_CONFIG`). Supported overrides:

| Setting | Environment variable |
|---|---|
| Listen address | `SENTINEL_ADDR` |
| Read/read-header/write/idle/shutdown timeouts | `SENTINEL_READ_TIMEOUT`, `SENTINEL_READ_HEADER_TIMEOUT`, `SENTINEL_WRITE_TIMEOUT`, `SENTINEL_IDLE_TIMEOUT`, `SENTINEL_SHUTDOWN_TIMEOUT` |
| Header/body limits | `SENTINEL_MAX_HEADER_BYTES`, `SENTINEL_MAX_BODY_BYTES` |
| JWT secret, issuer, audience | `SENTINEL_JWT_SECRET`, `SENTINEL_JWT_ISSUER`, `SENTINEL_JWT_AUDIENCE` |
| Rate rps, burst, client cap, idle TTL, cleanup interval | `SENTINEL_RATE_RPS`, `SENTINEL_RATE_BURST`, `SENTINEL_RATE_MAX_CLIENTS`, `SENTINEL_RATE_IDLE_TTL`, `SENTINEL_RATE_CLEANUP` |
| Breaker threshold/open duration | `SENTINEL_BREAKER_FAILURE_THRESHOLD`, `SENTINEL_BREAKER_OPEN_TIMEOUT` |
| Users URL/prefix/timeout/connection cap | `SENTINEL_USERS_URL`, `SENTINEL_USERS_PREFIX`, `SENTINEL_USERS_TIMEOUT`, `SENTINEL_USERS_MAX_CONNECTIONS` |
| Payments URL/prefix/timeout/connection cap | `SENTINEL_PAYMENTS_URL`, `SENTINEL_PAYMENTS_PREFIX`, `SENTINEL_PAYMENTS_TIMEOUT`, `SENTINEL_PAYMENTS_MAX_CONNECTIONS` |

### Build and tests

```powershell
go test -race ./...
go vet ./...
go build ./cmd/sentinel ./mocks/users ./mocks/payments
```

Run a load test after exporting a valid token:

```powershell
$env:JWT_TOKEN = "<generated-token>"
k6 run .\loadtest\k6.js
```

## Failure Drills

1. **Stop `payments-service`:** stop the payments container with
   `docker compose -f .\deploy\docker-compose.yml stop payments`. Send at least
   five authenticated requests to `/api/payments/42`. Proxy failures return
   502/504 as they accumulate; once the threshold is reached, the breaker opens
   and calls fail fast with 503. Check
   `sentinel_circuit_breaker_state{upstream="payments"}` at `/metrics`. After
   ten seconds, the first request is the half-open probe.
2. **Exceed the rate limit:** set `$env:SENTINEL_RATE_RPS = "1"` and
   `$env:SENTINEL_RATE_BURST = "2"`, recreate the Compose stack, then issue
   several requests quickly from the same client IP. The excess requests
   return 429 with `Retry-After: 1`; check
   `sentinel_rate_limit_rejected_total`. Clear those environment variables and
   recreate the stack to restore the defaults.
3. **Send SIGTERM under load:** run k6 in one terminal and stop the gateway
   gracefully (`docker compose -f .\deploy\docker-compose.yml stop -t 15 gateway`,
   or send SIGTERM to the local process). Observe `shutdown started`, in-flight requests completing,
   then `shutdown complete`. Requests that finish inside the 15-second drain
   window should complete without being dropped. No server can promise zero
   dropped in-flight work if clients or upstreams exceed the drain deadline;
   verify this drill using the k6 checks rather than assuming it.

## Interview defense notes

1. **Why token bucket instead of a sliding window?** It provides a controllable
   burst while enforcing a long-run rate, with constant state and inexpensive
   per-request operations. A sliding window is preferable when strict window
   semantics matter more than implementation and state cost.
2. **How do you avoid a thundering herd as a breaker recovers?** Half-open mode
   admits only one probe (`MaxRequests: 1`); other requests do not all hit the
   recovering dependency simultaneously.
3. **What happens at 10k RPS with the in-memory limiter?** A client gets one
   local bucket, while synchronization covers only bucket lookup and entries
   are bounded/expired. Limits are per process; replicas multiply the effective
   allowance unless a shared limiter is introduced.
4. **Why use a reverse proxy instead of copying request/response bodies?**
   `httputil.ReverseProxy` already implements HTTP forwarding and hop-by-hop
   header handling. Maintaining a custom proxy would add protocol edge cases.
5. **How is a circuit breaker failure defined?** Any response with status 500
   or above counts as a failure, including proxy-generated 502/504 responses;
   ordinary 4xx responses do not.
6. **Can one upstream take down the other?** No. Each route owns a separate
   breaker and each upstream has independent state; shared CPU or network
   exhaustion remains a host-level failure domain.
7. **How do you prevent metrics cardinality explosions?** Instrumentation uses
   fixed route labels rather than raw URL paths, request IDs, tokens, or user
   identifiers.
8. **What does graceful shutdown guarantee?** It stops accepting new work and
   gives active handlers up to the configured drain duration. It does not
   guarantee completion beyond that deadline or protect against process/host
   failure.
9. **How do you keep JWT verification from accepting algorithm confusion?**
   Parsing explicitly allows HS256 and the key callback verifies the concrete
   signing method before returning the HMAC secret. Issuer, audience, expiry,
   and subject are validated.
10. **How should this evolve for multiple gateway replicas?** Move rate-limit
    state to a shared store or use a deliberate per-replica quota strategy,
    distribute secrets through a secret manager, add telemetry exporters, and
    benchmark connection and CPU limits in the deployment environment.

## Dependency choices

- `github.com/prometheus/client_golang` supplies Prometheus collectors, a
  private registry, and the standards-compliant scrape handler.
- `github.com/sony/gobreaker` supplies a concurrency-safe closed/open/half-open
  circuit breaker.
- `golang.org/x/time/rate` implements the token-bucket limiter.
- `github.com/golang-jwt/jwt/v5` parses and validates HS256 JWTs with registered
  claims.
- `gopkg.in/yaml.v3` parses strict, configuration-driven YAML.

## Out of Scope & Future Work

- Distributed rate limiting, distributed breaker coordination, and persistent
  per-client quotas.
- Key rotation/JWKS, OAuth/OIDC flows, API-key management, and tenant-specific
  authorization policies.
- OpenTelemetry SDK/exporters, distributed traces, and span sampling. The
  gateway only forwards W3C trace context.
- Dynamic configuration reload, service discovery, TLS termination, and
  deployment-specific ingress policy.
- Benchmarks are intentionally not fabricated; measure and fill the table
  above on the intended hardware and workload.

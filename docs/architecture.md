# Architecture

## System overview

SENTINEL is a Go HTTP reverse proxy for two upstream services. The public
listener routes `/api/users/*` and `/api/payments/*` to independently configured
upstreams. The API path applies request-size protection, client-IP rate
limiting, HS256 JWT validation, per-upstream circuit breaking, route deadlines,
and Prometheus request instrumentation. `/healthz`, `/readyz`, and `/metrics`
are operational endpoints on the same listener. The proxy uses
`net/http/httputil.ReverseProxy`; there is no routing framework.

```text
Incoming request
  │
  ├─ Request ID (preserve or generate UUIDv4)
  ├─ Structured JSON logger
  ├─ Panic recovery
  │
  ├─ API routes
  │    ├─ Metrics instrumentation
  │    ├─ Route timeout context
  │    ├─ Upstream-specific circuit breaker
  │    ├─ JWT authentication
  │    ├─ Per-client-IP token bucket
  │    ├─ MaxBytesReader body cap
  │    └─ Prefix-stripping ReverseProxy
  │           ├─ users upstream
  │           └─ payments upstream
  │
  └─ Operational routes: /healthz, /readyz, /metrics
```

Operational endpoints receive the outer request ID, logger, and recovery
middleware, but not API authentication, rate limiting, API body limits, route
timeouts, or API request metrics.

## Components and ownership

| Component | Responsibility |
|---|---|
| `cmd/sentinel` | Load validated config, construct handlers, start the HTTP server, and drain on signals |
| `internal/config` | Strict YAML parsing, environment overrides, and configuration validation |
| `internal/proxy` | Build one reverse proxy, transport, and circuit breaker for each configured route |
| `internal/middleware` | Compose request ID, logging, recovery, body-limit, rate-limit, JWT, breaker, timeout, and metrics handlers |
| `internal/metrics` | Own a private Prometheus registry and the fixed-cardinality collectors |
| `internal/health` | Liveness and atomic readiness handlers |
| `mocks/users`, `mocks/payments` | Small JSON upstreams for local and Compose testing |

## Request behavior

1. The gateway preserves an incoming `X-Request-ID` up to 128 characters; it
   generates a UUIDv4 when missing or too long. The value is returned to the
   client and forwarded upstream.
2. The logger records method, URL path, status, elapsed milliseconds, request
   ID, and the socket peer IP. It does not record authorization headers.
3. API traffic is timed and routed by configured prefix. The circuit breaker
   wraps the authenticated proxy flow and is independent for each upstream.
4. JWT validation requires a Bearer token signed with HS256, a valid expiry,
   the configured issuer and audience, and a non-empty subject.
5. Authenticated requests consume a token from the client's in-memory
   per-IP bucket. Requests from new IPs are rejected once the configured client
   map is full.
6. The body is capped with `http.MaxBytesReader`. The reverse proxy strips the
   configured public prefix before sending the request upstream.
7. `ReverseProxy` handles hop-by-hop header removal and propagates context
   cancellation. `traceparent` and `tracestate` are passed through as a
   trace-context stub; SENTINEL does not create spans or export traces.

## Failure behavior

| Condition | Result |
|---|---|
| Missing or invalid JWT | 401; no token is consumed from the rate limiter |
| Rate budget exhausted or limiter client table full | 429 with `Retry-After: 1` |
| Declared body is too large | 413 before proxying |
| Streaming body exceeds limit while read by proxy | Proxy error handler returns 413 |
| Upstream returns 4xx | Upstream response is passed through; breaker treats it as a non-failure |
| Upstream returns 5xx | Response is passed through; breaker records a failure |
| Upstream connection fails | Proxy error handler returns 502; breaker records the failed status |
| Route deadline expires | Proxy error handler returns 504; the request context is canceled |
| Breaker is open or half-open capacity is occupied | Fast 503 |
| SIGINT/SIGTERM | Stop accepting new connections and drain active handlers up to the configured deadline |

The breaker counts HTTP responses with status 500 or higher, including
gateway-generated 502/504. It does not count network errors directly; those
errors are surfaced as 502 by `ReverseProxy` before the breaker sees the
response. Five consecutive failures open each breaker by default. After ten
seconds, the breaker enters half-open and allows one probe.

## State and scaling

Rate-limiter buckets and breaker states are process-local. Multiple gateway
replicas therefore enforce independent quotas and independent breaker
decisions. Use a distributed rate-limit store if a global quota is required;
breaker coordination should only be distributed if there is a measured need,
because each instance otherwise benefits from isolating its own observed
upstream health. Client bucket state is capped and idle entries are periodically
removed.

Backpressure is provided by each upstream transport's `MaxConnsPerHost` cap
(1,024 by default) and by route deadlines. Requests that reach the cap wait in
the transport for a connection; their route context bounds that wait. There is
no separate worker-pool queue.

## Design records

See [Architecture Decision Records](adr/README.md) for the rationale and
consequences of the proxy, limiter, breaker, and deployment choices.

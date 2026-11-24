# Observability

SENTINEL uses `log/slog` with a JSON handler and a private Prometheus registry.
The gateway does not initialize an OpenTelemetry SDK or exporter.

## Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `sentinel_http_requests_total` | Counter | `method`, `route`, `status` | API request count |
| `sentinel_http_request_duration_seconds` | Histogram | `method`, `route` | API handler time, seconds |
| `sentinel_rate_limit_rejected_total` | Counter | `route` | Requests rejected by per-IP limiter |
| `sentinel_circuit_breaker_state` | Gauge | `upstream` | `0` closed, `1` half-open, `2` open |
| `sentinel_upstream_errors_total` | Counter | `upstream` | 5xx responses seen inside breaker |

Route labels are fixed to `/api/users/*`, `/api/payments/*`, or `unmatched`;
raw URL paths, request IDs, IPs, and token claims are never Prometheus labels.
Request metrics are recorded for API routes only. Health and metrics scrapes are
not included in these API counters.

Histogram buckets (seconds): `0.001`, `0.0025`, `0.005`, `0.01`, `0.025`,
`0.05`, `0.1`, `0.25`, `0.5`, `1`, `2.5`, and `5`.

The breaker gauge initializes to closed at startup and updates on state
changes. The upstream error counter increments for responses with status 500
or higher before breaker state decisions.

## PromQL examples

Request rate by route and status:

```promql
sum by (route, status) (rate(sentinel_http_requests_total[5m]))
```

Approximate p99 API latency by route:

```promql
histogram_quantile(
  0.99,
  sum by (le, route) (rate(sentinel_http_request_duration_seconds_bucket[5m]))
)
```

Rate-limit rejections over five minutes:

```promql
sum by (route) (increase(sentinel_rate_limit_rejected_total[5m]))
```

Open breakers:

```promql
sentinel_circuit_breaker_state == 2
```

Upstream error rate:

```promql
sum by (upstream) (rate(sentinel_upstream_errors_total[5m]))
```

## Logs

Every request gets one JSON request log with:

- `method`, URL `path`, response `status`
- `latency_ms`
- `request_id`
- `client_ip` (socket peer IP)

The `Authorization` header and JWT token are not logged. Rejected request IDs
are also propagated to the response when request-ID generation succeeded.
Proxy failures and recovered panics emit separate error-level records; panic
records include a stack trace.

## Trace context

The reverse proxy forwards W3C `traceparent` and `tracestate` values when
present. This is context propagation only: there are no spans, sampling
decisions, exporter setup, or trace correlation guarantees in this version.

## Alerting and access

Useful initial signals include sustained upstream 5xx rate, any breaker open
for longer than the normal recovery window, 429 growth, elevated p99 latency,
and Prometheus scrape failure. Tune thresholds to the service SLO and workload.
Prometheus collection is unauthenticated at the application layer, so
firewall/ingress it to trusted monitoring systems.

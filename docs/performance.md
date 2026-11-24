# Performance and load testing

The repository includes a k6 ramping-VU scenario at `loadtest/k6.js`. It targets
the authenticated users route by default and progresses to 5,000 virtual users:

| Stage | Duration | Target VUs |
|---|---:|---:|
| Warm up | 30s | 100 |
| Ramp | 1m | 1,000 |
| Ramp | 1m | 2,500 |
| Sustained high load | 2m | 5,000 |
| Ramp down | 30s | 0 |

The script checks response success, fails if request error rate reaches 5%, and
sets a p99 request-duration threshold of one second. These are test thresholds,
not measured results or a throughput guarantee.

## Running the scenario

Start the gateway and upstreams with Compose. Provide a valid token in the
environment and run:

```powershell
$env:JWT_TOKEN = "<short-lived-token>"
k6 run .\loadtest\k6.js
```

Override the gateway endpoint with `BASE_URL` if needed. Do not put live tokens
in source control or shared benchmark artifacts.

## Measurement procedure

For a defensible benchmark:

1. Record CPU, memory, OS/kernel, Go version, container limits, and exact
   revision.
2. State whether gateway, mocks, Prometheus, and k6 run on the same host.
3. Start with a warm-up, then run repeatable duration-based tests. Capture
   k6's full summary, not only the highest observed RPS.
4. Record request mix, token validation, body size, upstream latency/error
   behavior, and whether the client is rate-limited.
5. Measure gateway process/container RSS and goroutine count at peak load.
   `docker stats` provides container memory/CPU; use an approved runtime
   profiling approach for goroutine counts.
6. Repeat at least three times and report median/variation. Do not claim a
   hardware-independent 10k-RPS capability from the configured limiter alone.

## Results template

| Metric | Value | Environment/revision |
|---|---|---|
| Throughput (RPS) | ___ | ___ |
| P50 latency | ___ | ___ |
| P95 latency | ___ | ___ |
| P99 latency | ___ | ___ |
| Error rate | ___ | ___ |
| Memory (RSS) | ___ | ___ |
| Goroutines at peak | ___ | ___ |

The root README includes the short-form benchmark table. Keep the environment
and revision alongside any values you publish.

## Capacity knobs

- Per-IP token rate and burst are configured in `rate_limit`.
- Per-upstream concurrent connections are capped by
  `routes.*.max_connections_per_host`.
- Route deadlines bound upstream waits and response times.
- Prometheus histogram buckets are tuned for millisecond-to-seconds API
  requests; change them only with a documented measurement/use case.
- Multiple gateway replicas multiply independent in-memory quotas.

Change one capacity variable at a time and verify latency, error rate, resource
use, and dependency saturation together.

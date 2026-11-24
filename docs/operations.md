# Operations runbook

## Start locally with Docker Compose

From the repository root in PowerShell:

```powershell
$env:PROMETHEUS_PORT = "9090" # choose a free host port if 9090 is occupied
docker compose -f .\deploy\docker-compose.yml up --build -d
docker compose -f .\deploy\docker-compose.yml ps
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
curl.exe http://localhost:8080/metrics
```

Stop and remove the demo stack:

```powershell
docker compose -f .\deploy\docker-compose.yml down
```

The default ports are gateway 8080 and Prometheus 9090. Upstreams are private
Compose services on ports 9001 and 9002. Prometheus scrapes
`gateway:8080/metrics` every five seconds.

## Health semantics

- `/healthz` returns 200 if the handler runs; it is a liveness signal, not an
  upstream dependency check.
- `/readyz` returns 200 during normal operation and 503 after shutdown begins.
  An upstream outage does not make the gateway unready.
- `/metrics` is unauthenticated for scraping; keep it private at ingress.

## Triage

1. Check `/healthz` and `/readyz`.
2. Check gateway container/process logs for JSON request entries and proxy
   errors. Correlate with `request_id`; authorization headers are not logged.
3. Check `sentinel_circuit_breaker_state` by upstream and the
   `sentinel_upstream_errors_total` counter.
4. Check `sentinel_http_requests_total` by route/status and rate-limit
   rejections.
5. Test the upstream from the gateway network. The Compose mocks are configured
   for quick isolated verification, not dependency readiness orchestration.
6. For slow-upstream symptoms, compare route timeout values, transport
   connection caps, and upstream saturation before increasing limits.

## Failure drills

### Upstream outage and circuit breaker

```powershell
docker compose -f .\deploy\docker-compose.yml stop payments
```

Send at least five authenticated requests to `/api/payments/42`. Proxy errors
return 502/504 while failures accumulate; after the fifth consecutive failure
the breaker opens and subsequent calls return 503. Check
`sentinel_circuit_breaker_state{upstream="payments"}` on `/metrics`. After the
configured open timeout (10 seconds by default), the breaker permits one
half-open probe. Restore the mock with:

```powershell
docker compose -f .\deploy\docker-compose.yml start payments
```

### Rate-limit rejection

Recreate the gateway with a deliberately low rate and burst:

```powershell
$env:SENTINEL_RATE_RPS = "1"
$env:SENTINEL_RATE_BURST = "2"
docker compose -f .\deploy\docker-compose.yml up --build -d --force-recreate gateway
```

After obtaining a valid token, make repeated requests from the same client IP.
Authenticated requests beyond the token budget return 429 and
`Retry-After: 1`. Check `sentinel_rate_limit_rejected_total`. The limiter is
after JWT auth, so failed authentication does not consume a bucket token.
Clear the overrides and recreate the gateway to restore Compose defaults:

```powershell
Remove-Item Env:SENTINEL_RATE_RPS -ErrorAction SilentlyContinue
Remove-Item Env:SENTINEL_RATE_BURST -ErrorAction SilentlyContinue
docker compose -f .\deploy\docker-compose.yml up --build -d --force-recreate gateway
```

### Graceful shutdown

Run k6 traffic while stopping the gateway:

```powershell
docker compose -f .\deploy\docker-compose.yml stop -t 15 gateway
docker compose -f .\deploy\docker-compose.yml logs --tail=30 gateway
```

Expect `shutdown started`, completed in-flight handlers, then `shutdown
complete` if the server drains before its configured deadline. The server
cannot guarantee zero dropped work when requests outlive the 15-second drain
window or the process/host is forcibly terminated.

## Production operational checklist

- Provide a strong secret using a secret manager; never use the checked-in
  development key.
- Put TLS termination and network policy in front of the gateway.
- Expose `/metrics` only to authorized scraping infrastructure.
- Set container CPU/memory limits and size the upstream connection caps and
  route deadlines from measured dependency capacity.
- Decide whether rate quotas should be per replica or globally shared.
- Configure orchestrator termination grace period to exceed the app drain
  duration and confirm probes stop routing before termination.
- Pin and scan base images; establish a rollbackable release procedure.

## Configuration and logs

See [configuration reference](configuration.md) for every YAML field and
environment override. Logs are JSON on stdout and include method, path, status,
latency, request ID, and socket peer address. No log rotation is needed inside
the process; configure log collection/retention at the runtime.

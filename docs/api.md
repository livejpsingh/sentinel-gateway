# HTTP API reference

All paths below are served by the gateway listener, default `:8080`. The
development Compose stack publishes that listener on port 8080.

## Routes

| Method | Path | Auth | Behavior |
|---|---|---|---|
| Any | `/api/users/*` | Bearer JWT required | Forwards to the users upstream |
| Any | `/api/payments/*` | Bearer JWT required | Forwards to the payments upstream |
| GET | `/healthz` | None | Returns `200 ok` while the process can serve the handler |
| GET | `/readyz` | None | Returns `200 ready`, or `503 not ready` after shutdown begins |
| GET | `/metrics` | None | Prometheus text exposition from the private registry |

Only the health paths are registered exactly. Unknown paths receive the
standard `net/http` mux 404 response. Metrics and probes are unauthenticated;
restrict access to them at the network boundary in non-local deployments.

## Proxy routing

The default public prefixes are `/api/users/` and `/api/payments/`. SENTINEL
strips `/api/users` or `/api/payments` and forwards the remaining path to the
configured upstream. For example:

```text
GET /api/users/42?detail=true
  → GET http://users-upstream/42?detail=true
```

The configured upstream URL may include a base path; `ReverseProxy` joins it
with the stripped request path. Query parameters and normal end-to-end headers
are forwarded. Hop-by-hop headers are handled by the standard library proxy.

## Authentication

Send a signed JWT in the `Authorization` header:

```http
Authorization: Bearer <JWT>
```

Accepted tokens use HS256 and must have a valid `exp`, the configured `iss` and
`aud`, and a non-empty `sub`. Invalid/missing credentials receive `401
Unauthorized` with `WWW-Authenticate: Bearer`. JWT claims are made available
to downstream handlers through request context, but no tenant authorization
policy is implemented.

## Request IDs and tracing headers

- `X-Request-ID`: preserved when non-empty and no longer than 128 characters;
  otherwise replaced with a generated UUIDv4.
- Response: the selected ID is returned as `X-Request-ID`.
- Upstream: the selected ID is forwarded in `X-Request-ID`.
- `traceparent` and `tracestate`: forwarded if present; they are not validated
  and no tracing SDK or exporter is configured.

## Error responses

| Status | Meaning |
|---|---|
| 401 | JWT missing, malformed, expired, wrong issuer/audience, or invalid signature |
| 413 | Request body exceeds configured maximum |
| 429 | Rate bucket exhausted or no bucket can be allocated; includes `Retry-After: 1` |
| 502 | Upstream connection/proxy failure |
| 503 | Circuit breaker is open or half-open request capacity is occupied |
| 504 | Upstream request exceeded the route timeout |

Upstream-generated status codes and bodies are passed through unless the
reverse proxy fails before it can return an upstream response.

## Example probes

```powershell
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
curl.exe http://localhost:8080/metrics
curl.exe -i -H "Authorization: Bearer $env:JWT_TOKEN" http://localhost:8080/api/users/42
```

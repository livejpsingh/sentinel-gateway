# Configuration reference

SENTINEL reads YAML at startup. The default path is `config/config.yaml`, which
can be changed with `SENTINEL_CONFIG` or the `-config` flag. Environment
overrides are applied after YAML parsing; invalid values or invalid resulting
configuration stop startup with an error.

## YAML fields and local defaults

| YAML key | Example default | Notes |
|---|---:|---|
| `server.address` | `:8080` | HTTP listen address |
| `server.read_timeout` | `10s` | Whole request read timeout |
| `server.read_header_timeout` | `5s` | Header read timeout |
| `server.write_timeout` | `10s` | Response write timeout |
| `server.idle_timeout` | `60s` | Keep-alive idle connection timeout |
| `server.shutdown_timeout` | `15s` | Maximum graceful-drain time |
| `server.max_header_bytes` | `1048576` | Maximum request header bytes |
| `server.max_body_bytes` | `1048576` | Maximum API request body bytes |
| `auth.secret` | Development-only placeholder | Minimum 32 bytes; replace outside local development |
| `auth.issuer` | `sentinel-local` | Required JWT issuer |
| `auth.audience` | `sentinel-api` | Required JWT audience |
| `rate_limit.requests_per_second` | `50000` | Per-process, per-client-IP refill rate |
| `rate_limit.burst` | `20000` | YAML token capacity; Compose overrides to `100000` |
| `rate_limit.max_clients` | `50000` | Maximum client-IP buckets kept at once |
| `rate_limit.idle_ttl` | `10m` | Bucket inactivity before cleanup removes it |
| `rate_limit.cleanup_interval` | `1m` | Idle-client cleanup ticker interval |
| `circuit_breaker.failure_threshold` | `5` | Consecutive 5xx/proxy failures to open |
| `circuit_breaker.open_timeout` | `10s` | Time before a single half-open probe is allowed |
| `routes.users.prefix` | `/api/users/` | Must be absolute, end in `/`, and not overlap the payments prefix |
| `routes.users.upstream` | `http://localhost:9001` | Absolute HTTP or HTTPS URL |
| `routes.users.timeout` | `3s` | Users route deadline |
| `routes.users.max_connections_per_host` | `1024` | Per-process upstream transport connection cap |
| `routes.payments.prefix` | `/api/payments/` | Must be absolute, end in `/`, and not overlap the users prefix |
| `routes.payments.upstream` | `http://localhost:9002` | Absolute HTTP or HTTPS URL |
| `routes.payments.timeout` | `5s` | Payments route deadline |
| `routes.payments.max_connections_per_host` | `1024` | Per-process upstream transport connection cap |

## Environment overrides

| Setting | Environment variable |
|---|---|
| Config file path | `SENTINEL_CONFIG` |
| Listen address | `SENTINEL_ADDR` |
| Server timeouts | `SENTINEL_READ_TIMEOUT`, `SENTINEL_READ_HEADER_TIMEOUT`, `SENTINEL_WRITE_TIMEOUT`, `SENTINEL_IDLE_TIMEOUT`, `SENTINEL_SHUTDOWN_TIMEOUT` |
| Header/body limits | `SENTINEL_MAX_HEADER_BYTES`, `SENTINEL_MAX_BODY_BYTES` |
| JWT values | `SENTINEL_JWT_SECRET`, `SENTINEL_JWT_ISSUER`, `SENTINEL_JWT_AUDIENCE` |
| Limiter rate and bucket capacity | `SENTINEL_RATE_RPS`, `SENTINEL_RATE_BURST` |
| Limiter client map and cleanup | `SENTINEL_RATE_MAX_CLIENTS`, `SENTINEL_RATE_IDLE_TTL`, `SENTINEL_RATE_CLEANUP` |
| Breaker threshold and open duration | `SENTINEL_BREAKER_FAILURE_THRESHOLD`, `SENTINEL_BREAKER_OPEN_TIMEOUT` |
| Users route | `SENTINEL_USERS_URL`, `SENTINEL_USERS_PREFIX`, `SENTINEL_USERS_TIMEOUT`, `SENTINEL_USERS_MAX_CONNECTIONS` |
| Payments route | `SENTINEL_PAYMENTS_URL`, `SENTINEL_PAYMENTS_PREFIX`, `SENTINEL_PAYMENTS_TIMEOUT`, `SENTINEL_PAYMENTS_MAX_CONNECTIONS` |

Durations use Go duration syntax, such as `1500ms`, `3s`, or `2m`. Integer
settings are parsed as base-10 integers. The rate is a floating-point
requests-per-second value.

## Deployment-specific values

Compose sets `SENTINEL_USERS_URL` and `SENTINEL_PAYMENTS_URL` to service DNS
names. It also defaults the Compose limiter to `50000` requests/second with a
`100000` burst, which differs from the YAML burst. `PROMETHEUS_PORT` changes
only the published host port; Prometheus still listens on 9090 inside Compose.

Do not store production secrets in committed YAML or container build arguments.
Inject `SENTINEL_JWT_SECRET` through a secret manager or protected runtime
environment. For Docker Compose demos, set the value in the shell before
starting the stack.

## Validation rules

The loader rejects unknown YAML fields. Required server timeouts, limits,
authentication settings, limiter values, breaker settings, both route records,
route timeouts, and connection limits must be positive/present. The JWT secret
must be at least 32 bytes. Upstreams must be absolute HTTP(S) URLs.

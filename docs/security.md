# Security notes and threat model

This capstone demonstrates useful gateway controls; it is not a complete
production security boundary. Production use requires deployment-specific
identity, TLS, secret-management, network, and authorization policy.

## Assets and trust boundaries

| Asset/boundary | Consideration |
|---|---|
| JWT signing secret | Must be secret, at least 32 bytes, and injected securely |
| API request bodies | Limited to a configurable maximum |
| Upstream services | Reachable from the gateway; request context cancels on timeout |
| Metrics and probes | Same listener, unauthenticated; isolate at network/ingress |
| Client identity | Derived from `RemoteAddr`, not caller-controlled forwarded headers |
| Logs | Include path and peer IP; exclude authorization token/header |

## Implemented controls

- JWT middleware accepts only HS256, validates signature, expiry, issuer,
  audience, and non-empty subject.
- Oversized declared request bodies are rejected and streamed bodies are
  wrapped in `http.MaxBytesReader`.
- `httputil.ReverseProxy` applies standard hop-by-hop header handling.
- `X-Request-ID` is bounded to 128 characters; long IDs are replaced.
- No request authorization header or token is included in structured request
  logs.
- Server header limits, read/write/idle timeouts, route deadlines, and
  per-upstream connection caps bound resource use.
- Limiter client state is capped and idle entries are periodically deleted.
- Distroless non-root runtime image reduces the container's installed tools and
  runs without root privileges.

## Important limitations

- The YAML secret is a local-development placeholder. Replace it in all
  deployed environments. Container images include the YAML file; never place a
  production secret in that file or image.
- HS256 has one shared secret and no key rotation/JWKS support. Use a managed
  identity provider and rotation strategy for production.
- JWT claims establish token validity only. The project does not implement
  role, scope, tenant, or resource-level authorization.
- The limiter keys on socket peer IP. When deployed behind a proxy, clients may
  appear to share one address. Forwarded headers are intentionally not trusted;
  only accept them after explicitly configuring trusted proxy ranges and
  validating the chain.
- Per-process rate buckets and circuit-breaker state are not shared across
  replicas.
- `/metrics`, `/healthz`, and `/readyz` are unauthenticated on the gateway
  listener. Restrict network exposure, especially for metrics.
- The local Compose stack exposes the JWT secret through its environment for
  demonstration convenience. Use Docker/Kubernetes secrets or an external
  secret manager in real deployments.
- No built-in TLS termination, CORS policy, request/response filtering, body
  redaction, audit trail, or OpenTelemetry exporter is included.

## Deployment recommendations

1. Terminate TLS at a trusted ingress or configure TLS explicitly.
2. Inject a high-entropy secret from an approved secret manager and rotate it
   deliberately; do not commit secrets or expose them in build args.
3. Restrict operational endpoints at network layer and use authenticated
   scrape access where required.
4. Define authorization rules beyond JWT signature validation.
5. Set resource limits, upstream egress allowlists, and per-service connection
   caps based on measured capacity.
6. Decide how client identity is established behind load balancers before
   enabling forwarded-IP trust.
7. Scan/pin container images and dependencies, review release provenance, and
   test rollback behavior.

# ADR 0002: Process-local rate-limit and breaker state

- Status: Accepted for the capstone
- Date: 2026-11-24

## Context

The project needs working per-client rate limiting and per-upstream circuit
breaking without adding a database or shared coordination service to the local
demo.

## Decision

Store token buckets and circuit-breaker state in memory within each gateway
process. Bound and expire limiter client entries. Use one breaker per upstream
per process.

## Consequences

- No network call or external state dependency is required per decision.
- State resets on restart and differs between replicas.
- Per-IP identity depends on the direct socket peer; forwarded headers are not
  trusted by default.
- A full client map rejects new identities until idle cleanup frees capacity.
- Each replica can independently permit a half-open breaker probe.

## Revisit when

Product requirements call for globally consistent quotas, durable rate state,
cross-replica breaker coordination, or a different trusted client identity.
Choose shared state based on measured requirements, availability tradeoffs,
latency budget, and failure policy.

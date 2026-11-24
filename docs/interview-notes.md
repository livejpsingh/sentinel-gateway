# Interview defense notes

These are concise answers to common senior-level questions. Discuss the
limitations explicitly rather than presenting the capstone as a finished
globally distributed platform.

1. **Why a token bucket?** It provides a configurable burst and steady refill
   rate with small, constant state. A sliding window is appropriate when exact
   window quotas matter more than burst tolerance and implementation/storage
   cost.
2. **How is the limiter scaled?** Buckets are process-local and keyed by socket
   peer IP. Replicas have independent quotas; use a shared store for global
   quotas or explicitly define per-replica budgets.
3. **What happens when the limiter map is full?** Existing clients continue to
   use their buckets; new client IPs are rejected with 429 until idle cleanup
   removes entries. A production system may need an eviction policy or shared
   limiter with explicit fairness guarantees.
4. **How does the breaker avoid a recovery herd?** It allows one half-open
   probe (`MaxRequests: 1`) after the open timeout. The breaker is per process,
   so several replicas may each probe once.
5. **What trips the breaker?** HTTP responses of 500 or above, including the
   gateway's 502/504 proxy errors, count as consecutive failures. 4xx responses
   are non-failures. This simple policy may need endpoint-specific treatment
   for real services.
6. **Why use `ReverseProxy`?** The standard library implements HTTP proxy
   details and hop-by-hop header processing. A custom body-copy proxy would
   recreate protocol behavior and edge cases unnecessarily.
7. **How are deadlines enforced?** A route-specific context deadline wraps the
   proxy call. Cancellation propagates to the upstream request; the error
   handler maps deadline expiration to 504.
8. **Does a valid JWT authorize every operation?** No. It verifies HS256
   signature, expiry, issuer, audience, and subject. Role/scope/tenant
   authorization is intentionally out of scope and must be designed separately.
9. **How is metric cardinality bounded?** Route labels come from a finite set;
   raw paths, user IDs, request IDs, and tokens are not labels.
10. **Why no worker pool?** The HTTP transport caps in-flight connections per
    upstream and route contexts bound queue wait. A second worker queue would
    add queueing latency and another capacity boundary without evidence it is
    needed.
11. **What does shutdown guarantee?** The server stops accepting new
    connections and drains handlers until the configured deadline. It cannot
    promise completion beyond that deadline or survive force termination.
12. **What changes for multi-region production?** Shared quota semantics,
    identity/key rotation, service discovery, cross-region failure policy,
    telemetry/exporters, autoscaling limits, TLS/network controls, and
    rollbackable deployment all need explicit design and load-test evidence.

## Tradeoffs to acknowledge

- In-memory rate and breaker state are not durable or coordinated.
- IP-based client identity is insufficient behind a proxy unless trust is
  configured carefully.
- Operational endpoints are unauthenticated at the app layer and need network
  protection.
- The included mocks and load scenario support demonstrations, not
  independently verified production capacity claims.
- The published image pipeline delivers artifacts but does not deploy them.

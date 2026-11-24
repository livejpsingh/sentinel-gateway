# Architecture Decision Records

ADRs explain decisions that shape more than one implementation detail. They
record the current choice, the context that motivated it, and the consequences
that a future change must revisit.

| ADR | Decision |
|---|---|
| [0001](0001-net-http-reverse-proxy.md) | Use the standard library HTTP server and reverse proxy |
| [0002](0002-process-local-resilience-state.md) | Keep rate-limit and breaker state process-local for the capstone |
| [0003](0003-bounded-upstream-concurrency.md) | Use transport connection caps and deadlines instead of an extra worker queue |

# ADR 0003: Bound upstream concurrency with transport caps and deadlines

- Status: Accepted
- Date: 2026-11-24

## Context

The gateway should protect upstreams and bound waiting work without adding a
custom worker-pool and queue implementation.

## Decision

Set `MaxConnsPerHost` on each upstream HTTP transport and enforce a route
context deadline. Let the standard HTTP transport queue connection waiters;
cancel when the route deadline expires.

## Consequences

- Active upstream connections are capped independently for each destination.
- Requests waiting for a connection consume handler resources until completion
  or deadline expiration.
- There is no explicit queue depth limit, admission policy, or queue metric.
- Connection caps and timeouts must be tuned against upstream capacity and
  observed latency.

## Revisit when

Measurements show the transport wait queue creates unacceptable memory or
latency behavior, or the product requires explicit admission control and
load-shedding semantics.

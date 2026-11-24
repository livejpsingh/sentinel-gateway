# ADR 0001: Standard library HTTP server and reverse proxy

- Status: Accepted
- Date: 2026-11-24

## Context

SENTINEL is a reverse-proxy capstone intended to demonstrate explicit Go
middleware composition and HTTP behavior without hiding request flow behind a
web framework.

## Decision

Use `net/http` for the server and middleware interfaces, and
`net/http/httputil.ReverseProxy` for upstream forwarding.

## Consequences

- Handler composition and route behavior remain visible in ordinary Go code.
- The standard reverse proxy owns hop-by-hop header behavior and request
  forwarding details.
- Configuration, middleware order, and error mapping remain application
  responsibilities.
- Framework-specific integrations are not available without adding them
  explicitly.

## Revisit when

The project has requirements that justify a routing abstraction, protocol
support beyond HTTP reverse proxying, or a framework-specific middleware
ecosystem.

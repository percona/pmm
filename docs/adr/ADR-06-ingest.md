# ADR-06: Ingest path

- Status: Accepted
- Date: 2026-09-24
- Tickets: PMM-15567, PMM-15590, PMM-15571, PMM-15572

## Context

PMM Server must accept OTLP from PMM clients without exposing an unauthenticated port. The prototype bound OTLP on `0.0.0.0:4317/4318` on the server and on every client, and allowed Viewers on `/otlp/`.

## Options

1. Expose the server's OTLP ports, gRPC and HTTP, directly to the network.
2. Accept OTLP from anyone with PMM credentials, through nginx.
3. Accept OTLP only from node collectors, over HTTP through nginx, with the Admin role.

## Decision

Option 3:
- The server collector listens on `127.0.0.1:4318`, over HTTP only, with no gRPC.
- nginx `/otlp/` proxies to it behind `auth_request` and requires the Admin role. pmm-agent's per-node service-account tokens have that role. There is no anonymous access.
- Clients send only through their node's collector. The node collector opens an OTLP receiver only when traces are turned on (PMM-15579), and binds `127.0.0.1` by default.
- PMM Server's own node collector sends to `127.0.0.1:4318` directly, because the server's pmm-agent has no token.

## Consequences

- No OTLP port listens on a non-loopback address unless an admin sets one.
- An instrumented application sends to its node's collector, never to the server.
- Shared credentials mean the server can't tell which node sent a record. Node identity is asserted by the node collector. ADR-18 adds verification for per-node tokens.
- While the server collector is stopped, `/otlp/` answers 502, and node collectors retry from their disk queues.

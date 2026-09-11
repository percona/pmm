# vmproxy Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [managed/AGENTS.md](../managed/AGENTS.md) (configures VictoriaMetrics scrape targets)

**vmproxy** is a lightweight, stateless HTTP reverse proxy for VictoriaMetrics. It intercepts requests, reads label filters from a configurable HTTP header, and injects them as `extra_filters[]` query parameters before forwarding to VictoriaMetrics. This enables **label-based access control (LBAC)** — restricting which metrics a user can query based on their role.

## Architecture

### Request Flow

```
Client (Grafana / API)
  → HTTP request with X-Proxy-Filter header
    → vmproxy (parses header, injects extra_filters[])
      → VictoriaMetrics (applies filters to all queries)
        → response proxied back to client
```

### Filter Mechanism

1. Client includes an HTTP header (default: `X-Proxy-Filter`)
2. Header value is a **base64-encoded JSON array** of filter strings
3. Example: `WyJlbnY9UUEiLCAicmVnaW9uPUVVIl0=` decodes to `["env=QA", "region=EU"]`
4. vmproxy strips any existing `extra_filters[]` params and replaces them with the header values
5. VictoriaMetrics applies these filters as label matchers to all queries
6. Multiple filters are combined with logical OR by VictoriaMetrics

### Security
- **Read-only allow-list** — only the VictoriaMetrics query endpoints in `readOnlyPaths` (plus
  `/api/v1/label/<name>/values`) are forwarded; everything else gets `403 Forbidden` and a warn
  log naming the path. This is a capability gate, not a filter: it holds whether or not access
  control is enabled. See [Why the allow-list lives here](#why-the-allow-list-lives-here).
- **Admin marker** — pmm-managed sets `X-Proxy-Admin` when it has authenticated the caller as an
  admin, and nginx forwards it. A marked request skips the allow-list entirely. It is not a
  credential: nginx overwrites the header on every location that can reach the proxy, so a client
  cannot supply one, and the proxy listens on loopback only. Grafana's data source is never
  marked — `/graph` requires no role, so pmm-managed never authenticates it.
- Invalid headers (bad base64 or JSON) return `412 Precondition Failed`
- `X-Forwarded-For` is stripped
- Missing `User-Agent` is set to empty

### Why the allow-list lives here

Grafana's Metrics data source points at vmproxy, and Grafana forwards whatever sub-path it is
given. Every route to that data source — `/graph/api/ds/query`, both
`/graph/api/datasources/proxy/` forms, and `/graph/api/datasources/uid/<uid>/resources/` — crosses
this proxy, so this is the one place that sees them all. A gate in pmm-managed would instead have
to enumerate Grafana's URL forms, which is what PMM-15379 showed to be fragile.

Admins reach snapshots and the rest of the admin surface either through the admin-gated
`/prometheus/*` nginx location, which goes straight to VictoriaMetrics without crossing this
proxy, or through the marker described above — which is why restricting a marked request would
remove capability without removing exposure. Metric ingestion bypasses the proxy entirely via the
exact-match `/victoriametrics/api/v1/write` location.

Adding a path here widens what any dashboard user can reach; check it is a read endpoint first.

## Configuration

CLI flags (using **Kong**):

| Flag | Default | Purpose |
|------|---------|---------|
| `--target-url` | `http://127.0.0.1:9090` | VictoriaMetrics backend URL |
| `--listen-address` | `127.0.0.1` | Listen address |
| `--listen-port` | `1280` | Listen port |
| `--header-name` | `X-Proxy-Filter` | HTTP header containing filters |
| `--debug` | `false` | Enable debug logging |

## Implementation Details

The proxy is built on `net/http/httputil.ReverseProxy` with a custom `Director` function:

1. **`director()`** / **`prepareRequest()`** — rewrites `req.URL` to target, sets Host and Authorization
2. **`failOnDisallowedPath()`** / **`isReadOnlyPath()`** — refuses any path outside the read-only allow-list with 403, before the request is proxied
3. **`failOnInvalidHeader()`** — if the filter header is present but malformed, returns 412
4. **Filter injection** — removes existing `extra_filters[]`, parses header, adds each filter as `extra_filters[]`

## Patterns and Conventions

### Do
- Keep the proxy stateless — no caching, no session state
- Validate header format before processing (fail fast on malformed input)
- Use `net/http/httputil.ReverseProxy` as the base
- Test both valid and invalid filter scenarios

### Don't
- Don't add business logic beyond the access policy above — the proxy transforms headers into query parameters and gates the reachable paths, nothing more
- Don't cache responses — VictoriaMetrics handles caching
- Don't modify response bodies

## Testing

- Unit tests: `proxy/proxy_test.go`
- Integration tests: `main_test.go`
- Run: `make test`

## Key Files to Reference

- `vmproxy/main.go` — entry point and configuration
- `vmproxy/proxy/proxy.go` — core proxy logic

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
- Invalid headers (bad base64 or JSON) return `412 Precondition Failed`
- `X-Forwarded-For` is stripped
- Missing `User-Agent` is set to empty

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
2. **`failOnInvalidHeader()`** — if the filter header is present but malformed, returns 412
3. **Filter injection** — removes existing `extra_filters[]`, parses header, adds each filter as `extra_filters[]`

## Patterns and Conventions

### Do
- Keep the proxy stateless — no caching, no session state
- Validate header format before processing (fail fast on malformed input)
- Use `net/http/httputil.ReverseProxy` as the base
- Test both valid and invalid filter scenarios

### Don't
- Don't add business logic — the proxy should only transform headers into query parameters
- Don't cache responses — VictoriaMetrics handles caching
- Don't modify response bodies

## Testing

- Unit tests: `proxy/proxy_test.go`
- Integration tests: `main_test.go`
- Run: `make test`

## Key Files to Reference

- `vmproxy/main.go` — entry point and configuration
- `vmproxy/proxy/proxy.go` — core proxy logic

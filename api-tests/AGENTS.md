# PMM API Tests Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api/AGENTS.md](../api/AGENTS.md) (API definitions and generated clients used by tests) · [managed/AGENTS.md](../managed/AGENTS.md) (server-side implementation being tested)

The `/api-tests` directory contains **integration tests** for PMM APIs. These tests run against a **live PMM Server** and verify end-to-end API behavior using the generated Go HTTP clients from Swagger specs.

## Architecture

### How Tests Work

```
Go test binary
  → Generated Swagger HTTP clients (from /api/*/json/client/)
    → HTTP/JSON requests to PMM Server (gRPC-Gateway)
      → pmm-managed processes request
        → Database / VictoriaMetrics / ClickHouse
```

Tests use the same generated API clients that pmm-admin uses, ensuring client library correctness alongside API behavior.

### Test Organization

Tests are grouped by API domain, mirroring the `/api` directory structure:

| Directory | API Domain | What's Tested |
|-----------|------------|---------------|
| `alerting/` | Alerting API | Template CRUD, rule creation |
| `backup/` | Backup API | Backup operations, storage locations |
| `inventory/` | Inventory API | Node/Service/Agent CRUD, listing, filtering |
| `management/` | Management API | Add/remove MySQL, PostgreSQL, MongoDB, etc. |
| `management/action/` | Actions API | Explain, PT summary |
| `management/services/` | Management Services | Agent management |
| `server/` | Server API | Version, auth, settings |
| `user/` | User API | User preferences |

## Running Tests

### Prerequisites
- Running PMM Server (use `make env-up` from repo root)
- `PMM_SERVER_URL` environment variable (format: `http://USERNAME:PASSWORD@HOST`)

### Commands

```bash
# From api-tests/ directory:
make run          # Run tests, produce JUnit XML report
make run-dev      # Run tests without JUnit output
make run-race     # Run tests with race detector

# Direct invocation:
go test ./... -pmm.server-url=http://admin:admin@127.0.0.1 -v

# From repo root:
make api-test
```

### Docker execution:
```bash
docker build -t pmm-api-tests .
docker run --network host -e PMM_SERVER_URL=http://admin:admin@127.0.0.1 pmm-api-tests
```

## Patterns and Conventions

### Do
- Make tests **idempotent** — tests must clean up after themselves
- Use helper functions in `helpers.go` for common setup (creating nodes, services, agents)
- Use `testify/assert` and `testify/require` for assertions
- Test both success and error paths (invalid input, not found, permission denied)
- Use the generated Swagger clients for API calls (same clients as pmm-admin)
- Test with the `t.Cleanup()` pattern to ensure resources are removed even on failure

### Don't
- Don't leave test resources behind — always clean up nodes, services, agents
- Don't assume a specific server state — tests should be self-contained
- Don't use gRPC directly — use the HTTP/JSON Swagger clients
- Don't hardcode the PMM Server URL — use the `-pmm.server-url` flag

### Test Structure Pattern

```go
func TestAddMySQL(t *testing.T) {
    t.Parallel()

    // Setup: create required resources
    node := helpers.AddGenericNode(t, "test-node")
    t.Cleanup(func() { helpers.RemoveNode(t, node.NodeID) })

    // Test: perform the operation
    result, err := client.Default.ManagementService.AddService(params)
    require.NoError(t, err)

    // Assert: verify the result
    assert.Equal(t, expected, result.Payload)

    // Cleanup is handled by t.Cleanup
}
```

## Key Files to Reference

- `api-tests/init.go` — test flag setup (`-pmm.server-url`)
- `api-tests/helpers.go` — shared helper functions for test setup/teardown
- `api-tests/Makefile` — run targets
- `api-tests/inventory/` — comprehensive examples of CRUD test patterns
- `api-tests/management/mysql_test.go` — reference for management API tests

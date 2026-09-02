# PMM API Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [managed/AGENTS.md](../managed/AGENTS.md) (server-side implementation) · [admin/AGENTS.md](../admin/AGENTS.md) (CLI client) · [api-tests/AGENTS.md](../api-tests/AGENTS.md) (integration tests)

The `/api` directory is the **single source of truth** for all PMM APIs. It contains Protocol Buffer (`.proto`) definitions that generate gRPC servers, gRPC-Gateway HTTP/JSON endpoints, validation code, OpenAPI/Swagger specs, and Go client libraries. Every other component in the monorepo consumes the types and clients generated from these definitions.

## Architecture

### Generation Pipeline

```
.proto files (source of truth)
  → protoc-gen-go           → *.pb.go (Go structs)
  → protoc-gen-go-grpc      → *_grpc.pb.go (gRPC server/client interfaces)
  → protoc-gen-grpc-gateway → *.pb.gw.go (HTTP/JSON gateway handlers)
  → protoc-gen-validate     → *.pb.validate.go (message validation)
  → protoc-gen-openapiv2    → *.swagger.json (OpenAPI specs)
  → swagger generate client → json/client/ (Go HTTP clients from Swagger)
```

### Tooling

- **Buf** (`buf.yaml`, `buf.gen.yaml`) — manages proto compilation, linting, and breaking change detection
- **go-swagger** — generates typed Go HTTP clients from Swagger specs
- Dependencies: `buf.build/envoyproxy/protoc-gen-validate`, `googleapis`, `grpc-gateway`

### Per-Domain Layout

Each domain directory typically contains:

```
domain/v1/
├── domain.proto                  # Proto source (SERVICE OF TRUTH)
├── domain.pb.go                  # Generated: Go structs
├── domain_grpc.pb.go             # Generated: gRPC server/client
├── domain.pb.gw.go               # Generated: gRPC-Gateway
├── domain.pb.validate.go         # Generated: validation
├── domain.swagger.json           # Generated: OpenAPI spec
└── json/
    └── client/                   # Generated: Go HTTP client (from Swagger)
        ├── domain_client.go
        └── ...
```

## Key API Services

| Service | Proto Location | Purpose |
|---------|---------------|---------|
| `ServerService` | `server/v1/` | Version, readiness, settings, updates |
| `NodesService` | `inventory/v1/` | Node CRUD (list, get, add, remove) |
| `ServicesService` | `inventory/v1/` | Service CRUD |
| `AgentsService` | `inventory/v1/` | Agent CRUD and logs |
| `ManagementService` | `management/v1/` | High-level add/remove for MySQL, PostgreSQL, MongoDB, etc. |
| `BackupService` | `backup/v1/` | Start/schedule backups, list artifacts |
| `LocationsService` | `backup/v1/` | Backup storage locations |
| `RestoreService` | `backup/v1/` | Restore operations |
| `QANService` | `qan/v1/` | Query analytics reports, metrics, labels, examples |
| `CollectorService` | `qan/v1/` | QAN data ingestion from agents |
| `AdvisorService` | `advisors/v1/` | Advisor checks execution and results |
| `AlertingService` | `alerting/v1/` | Alert templates and rules |
| `AccessControlService` | `accesscontrol/v1beta1/` | Role management |
| `RealtimeAnalyticsService` | `realtimeanalytics/v1/` | RTA sessions and queries |
| `AgentService` | `agent/v1/` | Bidirectional agent ↔ server stream |
| `AgentLocalService` | `agentlocal/v1/` | Agent local status and reload |
| `HAService` | `ha/v1beta1/` | HA cluster status |

## Versioning Convention

- **`v1`** — stable API, backward-compatible changes only
- **`v1beta1`** — beta API, may have breaking changes (e.g., `dump/v1beta1/`, `accesscontrol/v1beta1/`, `ha/v1beta1/`)

## REST Path Naming

PMM follows [AIP-122](https://google.aip.dev/122) and [AIP-136](https://google.aip.dev/136) for `google.api.http` paths.

- **Collection identifiers are `lowerCamelCase`** — `/v1/management/enrollmentTokens`, `/v1/advisors/failedServices`. Not kebab-case, not snake_case.
- **Custom methods use `:verb` in `lowerCamelCase`** — `/v1/inventory/services:getTypes`, `/v1/realtimeanalytics/sessions:start`.
- **Path parameters keep the proto field name**, which is `snake_case` — `/v1/management/nodes/{node_id}`.

Two deliberate departures from AIP-136:

- **A POST-based read keeps a `get` prefix** — `/v1/qan:getLabels`, `/v1/inventory/services:getTypes`. AIP-136 bars standard method verbs (`get`, `list`, …) from custom methods, but without the prefix `POST /v1/qan:labels` reads like a create. Marking the read intent wins over the rule here.
- **The URI verb is chosen for HTTP readability, not to mirror the RPC name** — `ListActiveServiceTypes` is exposed as `:getTypes`. AIP-136 requires the two to match; PMM optimizes the path for REST clients instead.

Nothing enforces this: `buf lint` checks proto identifiers, not the path strings inside `google.api.http` annotations. Two older paths predate the convention and are kebab-case (`/v1/backups/{artifact_id}/compatible-services` and `/v1/backups/artifacts/{artifact_id}/pitr-timeranges`). Don't copy them, and don't rename them either — a released path cannot change without breaking clients.

## Patterns and Conventions

### Do
- Edit only `.proto` files — they are the source of truth
- Run `make gen` (from repo root) after any proto change
- Use `go tool buf lint` to validate proto files before committing
- Add `(validate.rules)` annotations for field validation
- Use `google.api.http` annotations for REST endpoint mapping
- Use gRPC status codes (`codes.NotFound`, `codes.InvalidArgument`, etc.) not HTTP status codes
- Follow RESTful conventions for HTTP mappings (GET for reads, POST for creates, PUT for updates, DELETE for deletes)
- Name REST paths per [REST Path Naming](#rest-path-naming) above
- Add comments to proto messages and fields — they become API documentation

### Don't
- **Never edit generated files** (`*.pb.go`, `*.pb.gw.go`, `*.pb.validate.go`, `*.swagger.json`, `json/client/`)
- Don't introduce breaking changes to `v1` APIs (use `v1beta1` for experimental APIs)
- Don't add business logic to the API layer — it belongs in `managed/services/`
- Don't skip validation annotations on incoming request messages

## Code Generation Workflow

```bash
# From repo root — generates everything
make gen

# From api/ directory — generates API code only
make gen

# Lint proto files
make check   # runs buf lint

# Serve API docs locally
cd api && make serve
```

### What `make gen` Does
1. `go tool buf generate` — compiles `.proto` files to Go, gRPC, gateway, validation, OpenAPI
2. `swagger generate client` — generates typed Go HTTP clients per API domain
3. Formats generated code

## Testing

API definitions themselves are not unit-tested. Testing happens at:
- **Server-side**: `managed/services/*/` — unit tests for gRPC server implementations
- **Integration**: `/api-tests/` — tests against a live PMM Server using generated Go HTTP clients
- **Proto linting**: `go tool buf lint` catches style and compatibility issues

## Key Files to Reference

- `api/buf.yaml` — Buf configuration, dependencies, lint rules
- `api/buf.gen.yaml` — code generation plugin configuration
- `api/Makefile` — generation and serving targets
- `api/common/common.proto` — shared types used across all APIs
- `api/inventory/v1/` — the core inventory API (nodes, services, agents) is a good reference for API patterns
- `api/swagger/swagger.json` — merged OpenAPI spec

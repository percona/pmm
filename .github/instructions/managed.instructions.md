---
applyTo: managed/**
---
# pmm-managed Development Guidelines

> **Parent guide**: [PMM_AGENTS.md](../../PMM_AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api.instructions.md](api.instructions.md) (API definitions) · [agent.instructions.md](agent.instructions.md) (client agent) · [qan-api2.instructions.md](qan-api2.instructions.md) (QAN backend)

**pmm-managed** is the core backend service of PMM Server. It manages configuration of server-side components (VictoriaMetrics, Grafana, QAN, VMAlert, Alertmanager), maintains the inventory of monitored nodes/services/agents, orchestrates backups, runs advisor checks, handles HA consensus, and exposes gRPC/REST APIs consumed by pmm-admin, pmm-agent, and the UI.

## Architecture

### High-Level Design

```
pmm-admin (CLI) ──→ gRPC/REST API ──→ pmm-managed ──→ PostgreSQL (inventory, settings)
PMM UI ───────────→ gRPC-Gateway ──→                ──→ VictoriaMetrics (scrape config)
pmm-agent ────────→ bidirectional gRPC stream ──→    ──→ Grafana API (dashboards, users)
                                                     ──→ Supervisord (process management)
                                                     ──→ qan-api2 (QAN forwarding)
                                                     ──→ VMAlert (alerting rules)
```

### Service Architecture Pattern

Services follow a consistent dependency-injection pattern:

```go
type Service struct {
    db       *reform.DB
    l        *logrus.Entry
    // other dependencies as interfaces
}

func New(db *reform.DB, logger *logrus.Entry, ...) *Service {
    return &Service{db: db, l: logger, ...}
}
```

All services are composed and wired in `managed/cmd/pmm-managed/main.go`.

### API Layer

- **gRPC** (port 7771) — primary API protocol
- **REST/JSON** (port 7772) — gRPC-Gateway, auto-generated from proto definitions
- **Debug** (port 7773) — `/debug/metrics`, `/debug/pprof`, `/debug/vars`

gRPC server implementations live in `services/*/grpc/` subdirectories. They delegate to the parent service package for business logic.

## Directory Structure

```
managed/
├── cmd/
│   ├── pmm-managed/               # Main server binary — bootstrap, wiring, startup
│   ├── pmm-managed-init/          # Init/bootstrap helper
│   ├── pmm-managed-starlark/      # Starlark script runner for advisor checks
│   └── pmm-encryption-rotation/   # DB encryption key rotation tool
├── data/
│   ├── advisors/                  # Advisor definitions (YAML)
│   ├── checks/                    # Check definitions (MySQL, MongoDB, PostgreSQL)
│   └── alerting-templates/        # Alert rule templates
├── models/
│   ├── *_model.go                 # Domain model structs with reform tags
│   ├── *_reform.go                # Generated reform ORM code (DO NOT EDIT)
│   ├── *_helpers.go               # CRUD operations and query helpers
│   └── database.go                # Schema migrations (117+ versions), OpenDB, SetupDB
├── services/
│   ├── agents/                    # Agent registry, lifecycle, bidirectional gRPC handler
│   │   ├── grpc/                  # Agent gRPC server implementation
│   │   └── channel/               # Agent communication channel
│   ├── inventory/                 # Nodes, Services, Agents inventory CRUD
│   │   └── grpc/                  # Inventory gRPC servers
│   ├── management/                # High-level add/remove operations
│   │   ├── grpc/                  # Management gRPC servers
│   │   ├── backup/                # Backup management service
│   │   └── dump/                  # PMM dump service
│   ├── server/                    # Server settings, version, logs, updates
│   ├── backup/                    # Backup orchestration, compatibility, PBM PITR
│   ├── checks/                    # Advisor check execution (Starlark)
│   ├── alerting/                  # Alert template management
│   ├── victoriametrics/           # VictoriaMetrics scrape config generation
│   ├── vmalert/                   # VMAlert rules management
│   ├── grafana/                   # Grafana HTTP API client
│   ├── qan/                       # QAN API client (forwards to qan-api2)
│   ├── supervisord/               # Supervisord config generation and process control
│   ├── scheduler/                 # Scheduled tasks (backup schedules)
│   ├── ha/                        # High Availability (Raft consensus, gossip, leader election)
│   ├── telemetry/                 # Telemetry collection and reporting
│   ├── realtimeanalytics/         # Real-Time Analytics session management
│   ├── user/                      # User preferences service
│   ├── versioncache/              # Software version cache
│   ├── minio/                     # MinIO/S3 client for backup storage
│   └── config/                    # YAML config loader (deprecated)
├── pi/                            # Percona Intelligence integration
│   ├── alert/                     # Alert parameter handling
│   └── common/                    # Shared PI types
├── utils/
│   ├── envvars/                   # PMM_* environment variable parsing
│   ├── encryption/                # Database column-level encryption
│   ├── interceptors/              # gRPC middleware (auth, logging, error mapping)
│   ├── platform/                  # Percona Platform API client
│   ├── clean/                     # Cleanup jobs
│   └── distribution/              # Linux distribution detection
└── testdata/                      # Test fixtures (pg/, victoriametrics/, updater/)
```

## Domain Model

### Core Entities (PostgreSQL, reform ORM)

| Entity | Table | Model File | Description |
|--------|-------|------------|-------------|
| **Node** | `nodes` | `node_model.go` | Host or target: generic, container, remote, remote_rds, remote_azure_database |
| **Service** | `services` | `service_model.go` | DB/app: mysql, mongodb, postgresql, proxysql, haproxy, external, valkey |
| **Agent** | `agents` | `agent_model.go` | Monitoring agent: pmm-agent, exporters, QAN agents, vmagent, etc. |
| **Settings** | `settings` | `settings_model.go` | Server configuration (JSONB, singleton row) |
| **BackupLocation** | `backup_locations` | — | S3/local backup storage targets |
| **Artifact** | `artifacts` | — | Backup artifacts |
| **ScheduledTask** | `scheduled_tasks` | — | Scheduled backup tasks |
| **RestoreHistory** | `restore_history` | — | Backup restore records |
| **Role** | `roles` | — | Access control roles |

### Relationships

```
Node (1) ──→ (N) Service
Service (1) ──→ (N) Agent (via service_id)
Node (1) ──→ (N) Agent (via runs_on_node_id)
PMM Agent (1) ──→ (N) Child Agent (via pmm_agent_id)
```

### Database Layer (reform ORM)

PMM uses **reform** (NOT gorm) for PostgreSQL:

```go
//go:generate ../../bin/reform

//reform:nodes
type Node struct {
    NodeID   string `reform:"node_id,pk"`
    NodeName string `reform:"node_name"`
}
```

**Key conventions:**
- Models: `managed/models/*_model.go` with `//go:generate` directives
- Generated: `managed/models/*_reform.go` (never edit)
- CRUD helpers: `managed/models/*_helpers.go`
- Always accept `reform.Querier` parameter (works with both `*reform.DB` and `*reform.TX`)
- Check `reform.ErrNoRows` explicitly for "not found"
- Use `models.Find*()` helpers, not `q.Reload()` or `q.SelectOneFrom()` directly
- Transactions: `db.InTransactionContext(ctx, nil, func(tx *reform.TX) error { ... })`
- Schema migrations in `models/database.go` (`databaseSchema` map, versioned)

## Configuration

- **Environment variables**: `utils/envvars.ParseEnvVars()` parses `PMM_*` vars; `server.UpdateSettingsFromEnv()` persists to DB
- **Database settings**: `settings` table (JSONB); `models.GetSettings()`, `models.UpdateSettings()`
- **YAML config**: `services/config` loads `/etc/percona/pmm/pmm-managed.yml` (deprecated, mainly telemetry)
- **CLI flags**: Kingpin flags for PostgreSQL DSN, VictoriaMetrics URL, HA config, debug ports

## Key Packages

| Package | Responsibility |
|---------|---------------|
| `models` | Domain types, reform models, DB schema migrations, CRUD helpers |
| `services/agents` | Agent registry, bidirectional gRPC handler, state tracking |
| `services/inventory` | Nodes, Services, Agents CRUD with validation |
| `services/management` | High-level add/remove operations (combines inventory + agent setup) |
| `services/server` | Settings, version, update logic, logs |
| `services/backup` | Backup orchestration, compatibility checks, PBM PITR |
| `services/checks` | Advisor check execution via Starlark |
| `services/alerting` | Alert template management |
| `services/victoriametrics` | VictoriaMetrics scrape config generation from agent/service inventory |
| `services/vmalert` | VMAlert alerting rules generation |
| `services/grafana` | Grafana API client (users, dashboards, annotations) |
| `services/supervisord` | Supervisord config file generation and process control |
| `services/ha` | Raft consensus, gossip protocol, leader election |
| `services/telemetry` | Telemetry data collection and reporting to Percona |

## High Availability (HA)

PMM supports HA via **Raft consensus** (`services/ha/`):
- Distributed state using `hashicorp/raft`
- Agent states synchronized across nodes via gossip
- Leader election determines which node runs certain operations (e.g., scheduled backups)

## Patterns and Conventions

### Do
- Prefer modern Go idioms (context, error wrapping with `%w`)
- Use modern slice helpers (`slices.Contains`), range loops
- Use `any` instead of `interface{}`
- Define small interfaces in `deps.go` files for dependency injection and mocking
- Use `status.Error()` with proper gRPC codes for API errors
- Check `reform.ErrNoRows` for "not found" scenarios
- Wrap errors: `fmt.Errorf("descriptive context: %w", err)`
- Return early on errors to avoid deep nesting
- Use `errors.Is()` / `errors.As()` for error type checking
- Use standard `errors` package (not `github.com/pkg/errors`)
- Use structured logging: `s.l.WithField("key", value).Error("message")`
- Pass `*logrus.Entry` (not `*logrus.Logger`)
- Use RESTful conventions in proto HTTP annotations

### Don't
- Don't use `gorm` or other ORMs — only `reform`
- Don't edit generated files (`*_reform.go`, `*.pb.go`, `*.pb.gw.go`, swagger specs)
- Don't skip `make gen` after proto/model changes
- Don't comment on every line — only where clarity is needed
- Don't inline comments (`code // comment`) — put comments on separate lines
- Don't use named return values in functions
- Don't commit test binaries or artifacts
- Don't create subshells in Makefiles without reason

## Testing

### Unit Tests
- Use `testify/assert` and `testify/require`
- Mock generation via `mockery` (config in `.mockery.yaml`)
- Interface-based deps in `deps.go` files enable mocking
- `mock_*_test.go` files generated by mockery
- DB tests use `testdb` helper
- Run: `make test` (in managed/) or `make test-common` (from root)

### Integration Tests
- Located in `/api-tests/` (separate directory)
- Run against live PMM Server: `make api-test`

### Test Data
- `testdata/pg/` — PostgreSQL fixtures
- `testdata/victoriametrics/` — VictoriaMetrics configs
- `testdata/updater/` — Update test fixtures

## Code Generation

1. **Protocol Buffers** — `make gen` from repo root
2. **reform** — `//go:generate ../../bin/reform` on model files
3. **mockery** — mock generation per `.mockery.yaml`
4. **swagger** — API docs from proto annotations

Always run `make gen` after modifying `.proto` files, reform models, or interface signatures.

## Key Files to Reference

- `managed/cmd/pmm-managed/main.go` — application bootstrap, all service wiring
- `managed/models/database.go` — database schema and migrations
- `managed/models/node_model.go`, `service_model.go`, `agent_model.go` — core domain models
- `managed/services/agents/registry.go` — agent registration and lifecycle
- `managed/services/agents/grpc/agent_server.go` — bidirectional agent stream handler
- `managed/services/inventory/grpc/` — inventory API implementations
- `managed/services/ha/` — HA/Raft implementation
- `managed/utils/envvars/parser.go` — environment variable parsing
- `docker-compose.yml` — development environment
- `Makefile`, `Makefile.include` — build and development targets

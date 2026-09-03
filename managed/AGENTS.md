# pmm-managed Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api/AGENTS.md](../api/AGENTS.md) (API definitions) · [agent/AGENTS.md](../agent/AGENTS.md) (client agent) · [qan-api2/AGENTS.md](../qan-api2/AGENTS.md) (QAN backend)

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

Some gRPC server implementations live in `services/*/grpc/` subdirectories. They delegate to the parent service package for business logic. This is legacy, and newer services must implement gRPC directly in the service package.

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
//go:generate go tool reform

//reform:nodes
type Node struct {
    NodeID   string `reform:"node_id,pk"`
    NodeName string `reform:"node_name"`
}
```

**Key conventions:**
- Models: `managed/models/*_model.go`, each carrying a reform `//go:generate` directive; `make gen` regenerates the `*_reform.go` files
- Generated: `managed/models/*_reform.go` (never edit)
- CRUD helpers: `managed/models/*_helpers.go`
- Always accept `reform.Querier` parameter (works with both `*reform.DB` and `*reform.TX`)
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

Go style, error handling, logging, and code generation follow the [global conventions](../AGENTS.md#global-development-conventions) and are not repeated here. What follows is specific to pmm-managed.

### Do
- Define small interfaces in `deps.go` files for dependency injection and mocking

### Don't
- Don't connect to a real database in unit tests — use `github.com/DATA-DOG/go-sqlmock` to mock SQL queries; reserve `testdb.Open` for integration tests that genuinely require fixtures or migrations
- Don't use `gorm` or other ORMs — only `reform`
- Don't commit test binaries or artifacts
- Don't create subshells in Makefiles without reason

## Testing

### Unit Tests
- `mock_*_test.go` files are generated by mockery
- `go-sqlmock` wraps a `reform.DB`, so mocked queries are matched as SQL text
- Run: `make test` (in managed/) or `make test-common` (from root)

### Test Data
- `testdata/pg/` — PostgreSQL fixtures
- `testdata/victoriametrics/` — VictoriaMetrics configs

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
- `managed/CONTRIBUTING.md` — pmm-managed contribution notes (dev setup, patterns)

# PMM Development Guide for AI Agents

## Maintaining This Document

This file is read by every AI agent at session start. **You are responsible for keeping it accurate.** After completing work, check whether any of these apply:

- Added, removed, or renamed a top-level directory or component
- Added or removed a component guide in `.github/instructions/`
- Changed the tech stack (new dependency in `go.mod`, new tool, removed technology)
- Changed build targets in `Makefile` / `Makefile.include`
- Changed global conventions (code style, error handling, testing patterns)
- Changed architecture or data-flow (new pipeline, changed communication protocol)
- Changed the development environment (`docker-compose.yml`, `.devcontainer/`)

If any apply, update the relevant sections of this file. Also update the matching component guide in `.github/instructions/` if one exists for the affected area.

Do **not** update this file for routine code changes (bug fixes, minor feature implementation) that don't alter the repo's structure or conventions.

## How This Documentation Is Organized

This file is the **single authoritative entry point** for AI agents working with PMM. It provides the product-wide overview, architecture, domain model, conventions, and cross-links to component-specific guides.

### Component Guides

Each PMM component has a dedicated guide with architecture, directory structure, domain model, patterns, testing, and key files. When working on a specific component, read the relevant guide:

| Component | Guide | Scope |
|-----------|-------|-------|
| **pmm-managed** (server backend) | [managed.instructions.md](.github/instructions/managed.instructions.md) | `managed/**` |
| **pmm-agent** (client agent) | [agent.instructions.md](.github/instructions/agent.instructions.md) | `agent/**` |
| **pmm-admin** (CLI) | [admin.instructions.md](.github/instructions/admin.instructions.md) | `admin/**` |
| **APIs** (protobuf definitions) | [api.instructions.md](.github/instructions/api.instructions.md) | `api/**` |
| **qan-api2** (query analytics) | [qan-api2.instructions.md](.github/instructions/qan-api2.instructions.md) | `qan-api2/**` |
| **vmproxy** (VictoriaMetrics proxy) | [vmproxy.instructions.md](.github/instructions/vmproxy.instructions.md) | `vmproxy/**` |
| **UI** (React frontend) | [ui.instructions.md](.github/instructions/ui.instructions.md) | `ui/**` |
| **Dashboards** (Grafana dashboard definitions) | [dashboards.instructions.md](.github/instructions/dashboards.instructions.md) | `dashboards/dashboards/**` |
| **QAN App** (Grafana plugin & QAN panel) | [qan-app.instructions.md](.github/instructions/qan-app.instructions.md) | `dashboards/pmm-app/**` |
| **API Tests** (integration tests) | [api-tests.instructions.md](.github/instructions/api-tests.instructions.md) | `api-tests/**` |
| **Build & Packaging** | [build.instructions.md](.github/instructions/build.instructions.md) | `build/**` |

---

## Product Overview

Percona Monitoring and Management (PMM) is an open-source database monitoring solution for MySQL, MongoDB, PostgreSQL, ProxySQL, HAProxy, Valkey, and cloud databases (AWS RDS, Azure). It uses a **client-server architecture** where lightweight agents on monitored hosts collect metrics and query analytics data, sending them to a central server for storage, alerting, and visualization.

This is a **monorepository** containing multiple PMM components, APIs, documentation, and build scripts. Every backend component is written in Go; the UI is TypeScript/React.

## Architecture and Data Flow

### Metrics Pipeline

```
Exporters (node, mysqld, mongodb, postgres, proxysql, valkey, rds, azure)
  → VMAgent (scrapes exporters)
    → VictoriaMetrics (time-series storage on PMM Server)
      → Grafana (visualization)
      → VMAlert → Alertmanager (alerting)
```

### Query Analytics (QAN) Pipeline

```
QAN Agents (built into pmm-agent: perfschema, slowlog, pg_stat_statements, pg_stat_monitor, MongoDB profiler/log)
  → pmm-managed (gRPC receiver)
    → qan-api2 (gRPC collector)
      → ClickHouse (query analytics storage)
        → PMM UI / Grafana (visualization)
```

### Agent Communication

```
pmm-agent ←→ pmm-managed (bidirectional gRPC stream)
  - Server sends: SetStateRequest, StartAction, StartJob, Ping
  - Agent sends: StateChanged, QanCollect, ActionResult, JobResult, Pong
```

### Backup Pipeline

```
pmm-managed (orchestrator)
  → pmm-agent jobs (PBM for MongoDB, mysqldump/xtrabackup for MySQL)
    → S3/MinIO/local storage
```

## Domain Model

The core inventory model is **Node → Service → Agent**:

- **Node**: a physical or virtual host (generic, container, remote, RDS, Azure)
- **Service**: a database or application running on a node (MySQL, MongoDB, PostgreSQL, ProxySQL, HAProxy, Valkey, external)
- **Agent**: a monitoring agent associated with a node or service (pmm-agent, exporters, QAN agents, VMAgent)

Relationships:
- A Node has many Services
- A Service belongs to one Node
- An Agent runs on a Node (`runs_on_node_id`) and optionally monitors a Service (`service_id`)
- A child Agent belongs to a parent PMM Agent (`pmm_agent_id`)

## Repository Map

### Core Components

| Directory | Component | Purpose | Guide |
|-----------|-----------|---------|-------|
| `/managed` | pmm-managed | Server backend: inventory, APIs, VictoriaMetrics, Grafana, backup, alerting, HA | [managed.instructions.md](.github/instructions/managed.instructions.md) |
| `/agent` | pmm-agent | Client agent: exporters, QAN/RTA collectors, actions, backup/restore jobs | [agent.instructions.md](.github/instructions/agent.instructions.md) |
| `/admin` | pmm-admin | CLI for managing monitored services | [admin.instructions.md](.github/instructions/admin.instructions.md) |
| `/api` | APIs | Protobuf definitions and generated gRPC/REST/Swagger clients | [api.instructions.md](.github/instructions/api.instructions.md) |
| `/qan-api2` | qan-api2 | Query Analytics API: ClickHouse ingestion and analytics | [qan-api2.instructions.md](.github/instructions/qan-api2.instructions.md) |
| `/vmproxy` | vmproxy | VictoriaMetrics reverse proxy with LBAC filtering | [vmproxy.instructions.md](.github/instructions/vmproxy.instructions.md) |
| `/ui` | UI | React/TypeScript PMM frontend (Vite, MUI, TanStack Query) | [ui.instructions.md](.github/instructions/ui.instructions.md) |
| `/dashboards/dashboards` | Grafana Dashboards | Grafana dashboard JSON definitions for MySQL, MongoDB, PostgreSQL, OS, and more | [dashboards.instructions.md](.github/instructions/dashboards.instructions.md) |
| `/dashboards/pmm-app` | QAN App | Grafana application plugin bundling dashboards and the Query Analytics panel | [qan-app.instructions.md](.github/instructions/qan-app.instructions.md) |
| `/api-tests` | API Tests | Integration tests against live PMM Server | [api-tests.instructions.md](.github/instructions/api-tests.instructions.md) |
| `/build` | Build & Packaging | Docker, RPM/DEB, Packer, Ansible | [build.instructions.md](.github/instructions/build.instructions.md) |

### Supporting Directories

| Directory | Purpose |
|-----------|---------|
| `/docs` | API documentation and process docs (tech stack, best practices, git workflow) |
| `/documentation` | User-facing documentation (MkDocs) |
| `/tools` | Development tools (mockery, buf, golangci-lint, etc.) |
| `/version` | Version info and feature flags |
| `/dev` | Development utilities (e.g., mongo-rs-backups) |
| `/.devcontainer` | Devcontainer setup for local development |

### External Repositories

| Repository | Purpose                                                    |
|------------|------------------------------------------------------------|
| [percona/grafana](https://github.com/percona/grafana) | Percona's Grafana fork with PMM customizations             |
| [percona/node_exporter](https://github.com/percona/node_exporter) | Machine-level metrics exporter                       |
| [percona/mysqld_exporter](https://github.com/percona/mysqld_exporter) | MySQL server metrics exporter                    |
| [percona/mongodb_exporter](https://github.com/percona/mongodb_exporter) | MongoDB server metrics exporter                |
| [percona/postgres_exporter](https://github.com/percona/postgres_exporter) | PostgreSQL server metrics exporter           |
| [percona/proxysql_exporter](https://github.com/percona/proxysql_exporter) | ProxySQL server metrics exporter             |
| [percona/rds_exporter](https://github.com/percona/rds_exporter) | AWS RDS metrics exporter                               |
| [percona/azure_metrics_exporter](https://github.com/percona/azure_metrics_exporter) | Azure database metrics exporter    |
| [percona/pmm-qa](https://github.com/percona/pmm-qa) | End-to-end UI tests, QA automation DB setups and CLI tests         |
| [Percona-Lab/pmm-submodules](https://github.com/Percona-Lab/pmm-submodules) | Feature build orchestration                |

## Tech Stack

| Technology | Role |
|------------|------|
| **Go** | All backend components |
| **TypeScript / React** | PMM UI (`/ui`) |
| **Protobuf v3 / gRPC** | API definitions and inter-component communication |
| **grpc-gateway** | HTTP/JSON REST API generated from gRPC definitions |
| **PostgreSQL** | Primary data store for pmm-managed (inventory, settings, backups) |
| **ClickHouse** | Query analytics data store (qan-api2) |
| **VictoriaMetrics** | Time-series metrics storage |
| **VMAlert** | Alerting rules evaluation |
| **Grafana** | Dashboards and visualization |
| **reform** | Go ORM for PostgreSQL (used in pmm-managed only — NOT gorm) |
| **logrus** | Structured logging |
| **testify** | Test assertions (`assert`, `require` packages only — NOT suites) |
| **mockery** | Mock generation for Go interfaces |
| **golangci-lint** | Static analysis and linting |
| **Kong** | CLI framework for pmm-admin |
| **Docker Compose** | Development environment |
| **Ansible** | Server provisioning and configuration |
| **Packer** | Machine image builds (AMI, OVA, Azure, DigitalOcean) |

## Global Development Conventions

### Code Style
- Format with `gofumpt -s`; run `make format`
- Follow [Effective Go](https://golang.org/doc/effective_go.html) and [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments)
- Import grouping: stdlib, then external (`github.com/percona`, third-party), then internal (this repo)
- Use `any` instead of `interface{}`
- Use modern slice helpers (`slices.Contains`), range loops
- Don't use named return values
- Don't inline comments (`code // comment`); put comments on separate lines
- Don't add obvious/redundant comments; only comment non-obvious intent

### Error Handling
- Use `status.Error()` with proper gRPC codes for API errors
- Wrap errors with context: `fmt.Errorf("descriptive context: %w", err)`
- Return early on errors to avoid deep nesting
- Use `errors.Is()` and `errors.As()` for type checking
- Use standard `errors` package, not `github.com/pkg/errors`
- Check `reform.ErrNoRows` for "not found" scenarios in pmm-managed

### Logging
- Use `logrus` with structured fields
- Pass `*logrus.Entry` (not `*logrus.Logger`) to maintain context
- Format: `s.l.WithField("key", value).Error("message")`
- Log to unbuffered stderr; let the process supervisor handle the rest

### Environment Variables
- `PMM_DEV_*` — development/test only, never for end users
- `PMM_TEST_*` — not part of GA functionality
- `PMM_*` — GA functionality
- Use sub-prefixes for component groups (e.g., `PMM_HA_*`)

### Testing
- Use `testify/assert` and `testify/require` (not testify suites)
- Mock generation via `mockery` (config in `.mockery.yaml`)
- Unit tests: `*_test.go` next to implementation
- Integration tests: `/api-tests/`, run against live PMM Server
- E2E tests: [pmm-qa](https://github.com/percona/pmm-qa)

### Code Generation
- Protobuf/gRPC: `make gen` from repo root
- reform ORM: `//go:generate ../../bin/reform` (pmm-managed only)
- Mocks: `mockery` per `.mockery.yaml`
- **Never edit generated files** (`.pb.go`, `.pb.gw.go`, `*_reform.go`, `*.pb.validate.go`, swagger specs, `json/client/`)

### Graceful Shutdown
- Handle `SIGTERM` and `SIGINT` by canceling parent context
- Stop handling signals after first receipt so second signal terminates immediately
- Startup errors are fatal; runtime errors are handled, logged, and communicated

### Debug Endpoints
All long-running daemons expose on `127.0.0.1`:
- `/debug/metrics` — Prometheus metrics
- `/debug/vars` — expvar (command line, memory stats)
- `/debug/requests`, `/debug/events` — trace facility
- `/debug/pprof` — profiling

## Key Make Targets

| Target | Purpose |
|--------|---------|
| `make env-up` | Start development container (PMM Server) |
| `make env-up-rebuild` | Rebuild development container from scratch |
| `make gen` | Generate all code (protobuf, reform, mocks, format) |
| `make check` | Run linters (buf, golangci-lint, go-sumtype) |
| `make format` | Format code (gofumpt, goimports, gci) |
| `make release` | Build all binaries (agent, admin, managed, qan-api2) |
| `make test-common` | Run common unit tests |
| `make api-test` | Run API integration tests |
| `make prepare-pr` | Full pre-PR pipeline: gen + check-all + go mod tidy |

## Key Files to Reference

- `Makefile`, `Makefile.include` — build and development targets
- `docker-compose.dev.yml` — development environment (PMM Server, renderer, watchtower)
- `docker-compose.yml` — community/quickstart compose (stable image, minimal config)
- `go.mod` — Go module definition
- `.golangci.yml` — linter configuration
- `.mockery.yaml` — mock generation configuration
- `docs/process/tech_stack.md` — technology choices and rationale
- `docs/process/best_practices.md` — coding best practices
- `docs/process/GIT_AND_GITHUB.md` — git workflow

---

# Datacosmos fork — additions

This is the `datacosmos-br/pmm` fork. The sections above are upstream Percona's.
The rules below are datacosmos-specific and apply on the `datacosmos/build` branch.

## §W-9 — Multi-agent coordination (MANDATORY)

Multiple AI agents work this repo concurrently, **without git worktrees**
(shared checkout). Coordination is mandatory and standardized:

- **Protocol:** `.agents/skills/agent-coordination/SKILL.md`
- **Live ledger:** `.agents/coordination/LEDGER.md` — SSOT for agent identity +
  heartbeats, area locks (token strategy with TTL/lease), tasks with timestamps
  and expiry (vencimento), and the append-only communication log.

Every agent, every session: `git pull`; read the ledger; register/refresh your
Agents row; scan for STALE agents/locks/tasks; acquire a narrow Area Lock
before editing; heartbeat ≤15 min; small batches (one task → commit → next);
on stop, release locks + write a handoff Log entry. Never edit an area locked
by a live agent; never revert another agent's commits (open a `CONFLICT` task,
`owner: HUMAN`). Timestamps are UTC ISO-8601 (`date -u +%Y-%m-%dT%H:%M:%SZ`).

## Branch model

| Branch | Base | Purpose |
|---|---|---|
| `v3` | upstream `v3.7.1` | pristine upstream mirror — never edited directly |
| `datacosmos/build` | `v3.7.1` | datacosmos OCI/RPM packaging (`Makefile.datacosmos`) + this protocol |
| `feat/clickhouse-collector` | `v3.7.1` | isolated ClickHouse collector draft (not upstream-ready) |

Custom commits are never placed on `v3`, so `git merge upstream/v3` stays
conflict-free. Build details: `docs/datacosmos/BUILD.md`.

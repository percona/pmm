# PMM Development Guide for AI Agents

## Maintaining This Document

This file is read by every AI agent at session start. **You are responsible for keeping it accurate.** After completing work, check whether any of these apply:

- Added, removed, or renamed a top-level directory or component
- Added or removed a per-component `AGENTS.md`
- Changed the tech stack (new dependency in `go.mod`, new tool, removed technology)
- Changed build targets in `Makefile` / `Makefile.include`
- Changed global conventions (code style, error handling, testing patterns)
- Changed architecture or data-flow (new pipeline, changed communication protocol)
- Changed the development environment (`docker-compose.yml`, `.devcontainer/`)

If any apply, update the relevant sections of this file. Also update the matching per-component `AGENTS.md` if one exists for the affected area.

Do **not** update this file for routine code changes (bug fixes, minor feature implementation) that don't alter the repo's structure or conventions.

## How This Documentation Is Organized

This file is the **single authoritative entry point** for AI agents working with PMM. It provides the product-wide overview, architecture, domain model, conventions, and cross-links to component-specific guides.

### Component Guides

Each PMM component has a dedicated guide with architecture, directory structure, domain model, patterns, testing, and key files. When working on a specific component, read the relevant guide:

| Component | Guide | Scope |
|-----------|-------|-------|
| **pmm-managed** (server backend) | [managed/AGENTS.md](managed/AGENTS.md) | `managed/**` |
| **pmm-agent** (client agent) | [agent/AGENTS.md](agent/AGENTS.md) | `agent/**` |
| **pmm-admin** (CLI) | [admin/AGENTS.md](admin/AGENTS.md) | `admin/**` |
| **APIs** (protobuf definitions) | [api/AGENTS.md](api/AGENTS.md) | `api/**` |
| **qan-api2** (query analytics) | [qan-api2/AGENTS.md](qan-api2/AGENTS.md) | `qan-api2/**` |
| **vmproxy** (VictoriaMetrics proxy) | [vmproxy/AGENTS.md](vmproxy/AGENTS.md) | `vmproxy/**` |
| **UI** (React frontend) | [ui/AGENTS.md](ui/AGENTS.md) | `ui/**` |
| **Dashboards** (Grafana dashboard definitions) | [dashboards/dashboards/AGENTS.md](dashboards/dashboards/AGENTS.md) | `dashboards/dashboards/**` |
| **QAN App** (Grafana plugin & QAN panel) | [dashboards/pmm-app/AGENTS.md](dashboards/pmm-app/AGENTS.md) | `dashboards/pmm-app/**` |
| **API Tests** (integration tests) | [api-tests/AGENTS.md](api-tests/AGENTS.md) | `api-tests/**` |
| **Build & Packaging** | [build/AGENTS.md](build/AGENTS.md) | `build/**` |

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
| `/managed` | pmm-managed | Server backend: inventory, APIs, VictoriaMetrics, Grafana, backup, alerting, HA | [managed/AGENTS.md](managed/AGENTS.md) |
| `/agent` | pmm-agent | Client agent: exporters, QAN/RTA collectors, actions, backup/restore jobs | [agent/AGENTS.md](agent/AGENTS.md) |
| `/admin` | pmm-admin | CLI for managing monitored services | [admin/AGENTS.md](admin/AGENTS.md) |
| `/api` | APIs | Protobuf definitions and generated gRPC/REST/Swagger clients | [api/AGENTS.md](api/AGENTS.md) |
| `/qan-api2` | qan-api2 | Query Analytics API: ClickHouse ingestion and analytics | [qan-api2/AGENTS.md](qan-api2/AGENTS.md) |
| `/vmproxy` | vmproxy | VictoriaMetrics reverse proxy with LBAC filtering | [vmproxy/AGENTS.md](vmproxy/AGENTS.md) |
| `/ui` | UI | React/TypeScript PMM frontend (Vite, MUI, TanStack Query) | [ui/AGENTS.md](ui/AGENTS.md) |
| `/dashboards/dashboards` | Grafana Dashboards | Grafana dashboard JSON definitions for MySQL, MongoDB, PostgreSQL, OS, and more | [dashboards/dashboards/AGENTS.md](dashboards/dashboards/AGENTS.md) |
| `/dashboards/pmm-app` | QAN App | Grafana application plugin bundling dashboards and the Query Analytics panel | [dashboards/pmm-app/AGENTS.md](dashboards/pmm-app/AGENTS.md) |
| `/api-tests` | API Tests | Integration tests against live PMM Server | [api-tests/AGENTS.md](api-tests/AGENTS.md) |
| `/build` | Build & Packaging | Docker, RPM/DEB, Packer, Ansible | [build/AGENTS.md](build/AGENTS.md) |

### Supporting Directories

| Directory | Purpose |
|-----------|---------|
| `/docs` | API documentation and process docs (tech stack, best practices, git workflow) |
| `/documentation` | User-facing documentation (MkDocs) |
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
- Use `errors.Is()`, `errors.As()` or `errors.AsType()` for error inspection
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
- reform ORM: `//go:generate go tool reform` (pmm-managed only)
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
- `dev/docs/process/tech_stack.md` — technology choices and rationale
- `dev/docs/process/best_practices.md` — coding best practices
- `dev/docs/process/GIT_AND_GITHUB.md` — git workflow

## Cursor Cloud specific instructions

The dev environment is Docker-based: a single `pmm-server` dev container (image
`perconalab/pmm-server:3-dev-container`) bundles all server-side services
(pmm-managed, qan-api2, vmproxy, Grafana, VictoriaMetrics, ClickHouse, PostgreSQL,
nginx) under supervisord, with the repo bind-mounted at
`/root/go/src/github.com/percona/pmm`. Docker (with `fuse-overlayfs` storage and
`iptables-legacy`), Go, Node 22, and Yarn 1.22.22 are preinstalled in the VM
snapshot. The startup update script runs `yarn --cwd ui install` only.

Non-obvious caveats for starting/running things here:

- **Docker daemon is not auto-started.** Start it once per session and make the
  socket usable, e.g. run `sudo dockerd` in a background/tmux session, then
  `sudo chmod 666 /var/run/docker.sock` (the `ubuntu` user is in the `docker`
  group). Nothing below works until `docker ps` succeeds.
- **`.env` is required and gitignored.** Before `make env-up`, ensure it exists:
  `cp .env.dev.example .env`. It selects the dev-container image and dev settings.
- **Start PMM Server:** `make env-up` (wraps `docker compose -f docker-compose.dev.yml up -d --wait`).
  UI/API is at `https://localhost/` (self-signed cert) with `admin`/`admin`.
  Check health: `curl -sk https://localhost/v1/server/readyz`.
- **Rebuild Go services from source must run as root.** The documented
  `make env TARGET=run-managed` runs as the container's `pmm` user, but the fresh
  `go-modules`/`go-cache` Docker volumes are root-owned, so builds fail with
  `permission denied`. Build as root instead, e.g.
  `docker exec --user root --workdir=/root/go/src/github.com/percona/pmm pmm-server make run-managed-ci`
  (similarly `run-agent-ci`, `run-qan-ci`, `run-vmproxy-ci`). The first root build
  also needs `docker exec --user root pmm-server git config --global --add safe.directory /root/go/src/github.com/percona/pmm`
  to avoid git "dubious ownership" errors during VCS stamping. `go.mod` pins Go
  1.26.x while the container ships 1.25.x, so the first build auto-downloads the
  toolchain into the (root-owned) module cache — another reason to build as root.
- **Lint/test inside the container as root** to use the populated caches, e.g.
  `docker exec --user root --workdir=/root/go/src/github.com/percona/pmm pmm-server bash -lc 'go test ./vmproxy/...'`.
  `golangci-lint` v2.12.2 is on the container PATH (`make check` lints only new
  code via `--new-from-rev`).
- **`pmm-admin` inside the container cannot add services via the management API.**
  Access control is enabled (`PMM_ENABLE_ACCESS_CONTROL=1`) and `pmm-admin` has no
  server credentials, so `pmm-admin add ...` returns `401 Unauthorized`. Add
  services through the web UI (Add Instance) or the authenticated REST API
  (`curl -u admin:admin -X POST https://localhost/v1/management/services ...`).
  The internal PostgreSQL is reachable at `127.0.0.1:5432` as `pmm-managed`/`pmm-managed`.
- **UI dev server runs on the host, not in the container:** `cd ui && make dev`
  starts Vite on port 5173 (base path `/pmm-ui`); it is consumed through the PMM
  Server nginx proxy. `make build` and `make test` (vitest) also run on the host.

# PMM Development Guide for AI Agents

## How AI tools load this document

This file is the **single authoritative entry point** for AI agents. Tools are wired to it as follows:

| Tool | Entry file |
|------|------------|
| **Cursor** | `.cursor/rules/pmm-agents-entrypoint.mdc` (`alwaysApply: true`) → read this file |
| **Claude Code** | [CLAUDE.md](CLAUDE.md) → read this file |
| **GitHub Copilot** | [.github/copilot-instructions.md](.github/copilot-instructions.md) → read this file |

Local-only AI skills under `.claude/` and other `.cursor/` paths remain gitignored for personal experimentation.

## Maintaining This Document

**You are responsible for keeping this file accurate.** After completing work, check whether any of these apply:

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

This guide provides the product-wide overview, architecture, domain model, conventions, and cross-links to component-specific guides.

| Audience | Location |
|----------|----------|
| Human contributors | [`CONTRIBUTING.md`](CONTRIBUTING.md), [`docs/process/`](docs/process/) |
| AI agents | This file + component `AGENTS.md` guides |

This file **summarizes and links** process docs; it does not replace them. Pull out operational rules here only when agents routinely get them wrong.

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

## How AI agents should work in this repo

The **`AGENTS.md` hierarchy is the single source of truth** for agents. Read the matching component guide before editing.

Follow the sections through [Git and pull request checklist](#git-and-pull-request-checklist) every time you change code. Skim [Product Overview](#product-overview) and below when you need context.

### Workflow

1. Identify which component your change touches (`managed`, `ui`, `api`, `agent`, …).
2. Read that component's `AGENTS.md` before planning or editing.
3. Prefer minimal, focused diffs; match surrounding style and patterns.
4. After `.proto` or reform model changes: run `make gen` from the repo root.
5. Run the **smallest test set** that covers your change (see [Testing decision tree](#testing-decision-tree)).
6. Run the **matching linter** before calling work PR-ready (see [Linting decision tree](#linting-decision-tree)). For Go/API-only changes, step 7 covers this.
7. For Go/API changes: run `make prepare-pr` (gen + license check + Go lint + format + `go mod tidy` — subsumes step 6 for Go).
8. Update `AGENTS.md` (and the component guide) only if you changed structure, conventions, or workflows.

### Don'ts

- Don't edit generated files (`.pb.go`, `.pb.gw.go`, `*_reform.go`, `*.pb.validate.go`, swagger specs, `json/client/`).
- Don't use `gorm` in pmm-managed — **reform only**.
- Don't amend/squash commits locally to address review feedback; push **new commits** ([`docs/process/GIT_AND_GITHUB.md`](docs/process/GIT_AND_GITHUB.md)).
- Don't force-push to `main`/`v3`.
- Don't skip the Feature Build link in PR descriptions for user-facing changes ([`.github/pull_request_template.md`](.github/pull_request_template.md)).
- Don't run the full repo linter on every tiny edit; do run the **targeted linter** for what you changed, and run `make prepare-pr` before declaring Go/API work PR-ready.
- Don't write unit tests that call external services — use mocks or `/api-tests/` instead.

---

## PMM-specific choices (agents often get wrong)

These differ from generic Go/React advice. Match **surrounding code** in the file you edit; when in doubt, follow the component guide.

- **DB (managed):** reform only — never gorm or other ORMs ([`managed/AGENTS.md`](managed/AGENTS.md))
- **Unit tests (managed):** `go-sqlmock` by default — use `testdb.Open` only when migrations or fixtures are what you're testing ([`managed/AGENTS.md`](managed/AGENTS.md))
- **API errors (Go):** `status.Error()` with gRPC codes — not ad-hoc HTTP errors in service layers
- **Logging (Go):** `logrus` with `*logrus.Entry` and structured fields — not `fmt.Printf`
- **Mocks (Go):** small interfaces in `deps.go` + mockery — not hand-rolled fakes for every dependency
- **UI server state:** TanStack Query hooks in `ui/apps/pmm/src/hooks/api/` — not `useEffect` + `fetch` in components ([`ui/AGENTS.md`](ui/AGENTS.md))
- **UI client state:** React Context for auth/settings — not Redux or another global store
- **UI components:** MUI + `@percona/percona-ui`, theme-aware `sx` — not ad-hoc CSS
- **UI wire format:** camelCase in TypeScript; JSON on the wire is snake_case (`axios-case-converter` in `ui/apps/pmm/src/api/api.ts`)
- **Generated code:** edit `.proto` / reform models / interfaces — run `make gen`; never hand-edit `*.pb.go`, `*_reform.go`, swagger clients

Mechanical style (imports, formatting, ESLint rules) is enforced by `make check`, `cd ui && make lint`, and CI — see [Linting decision tree](#linting-decision-tree).

---

## Testing decision tree

PMM has three test layers ([`CONTRIBUTING.md`](CONTRIBUTING.md)): unit, API integration, and e2e (in [pmm-qa](https://github.com/percona/pmm-qa)). Use the smallest scope that validates your change:

| If you changed… | Run |
|-----------------|-----|
| Go unit logic in one package | `go test ./path/to/pkg/...` or `make test` in that component directory |
| Shared/API packages (not managed/admin/agent) | `make test-common` from repo root |
| `managed/models` or DB schema/migrations | Unit tests in `managed/`; use `testdb.Open` only when fixtures or migrations matter ([`managed/AGENTS.md`](managed/AGENTS.md)) |
| `.proto` or gRPC/REST definitions | `make gen`, then `make check`; update handlers in `managed/` and UI hooks if user-facing |
| REST behavior end-to-end | `make env-up`, then `make api-test` ([`api-tests/AGENTS.md`](api-tests/AGENTS.md)) |
| UI (`ui/apps/pmm`) | `cd ui && make lint && make test` |
| Grafana dashboard JSON (`dashboards/dashboards/`) | `python3 dashboards/misc/cleanup-dash.py --check-only <file>` (or run cleanup without `--check-only`); CI enforces this in `dashboards.yml` ([`dashboards/dashboards/AGENTS.md`](dashboards/dashboards/AGENTS.md)) |
| User-visible feature / bugfix | Create or update a Feature Build; link it in the PR ([`CONTRIBUTING.md`](CONTRIBUTING.md#feature-build)) |

---

## Linting decision tree

CI runs separate linters per area. `make prepare-pr` covers **Go only** — it does not lint UI or dashboards.

| If you changed… | Run |
|-----------------|-----|
| Go backend (`managed/`, `agent/`, `admin/`, `qan-api2/`, `vmproxy/`, shared packages) | `make prepare-pr` from repo root (or `make check` after `make gen` for a quicker pass) |
| `.proto` only | `make gen`, then `make check` (`buf lint`, `golangci-lint`, `go-sumtype`) |
| UI (`ui/apps/pmm`, `ui/packages/shared`) | `cd ui && make lint` (ESLint; same as CI `ui.yml`) |
| Grafana dashboard JSON (`dashboards/dashboards/`) | `python3 dashboards/misc/cleanup-dash.py --check-only <file>` before commit (CI `dashboards.yml`; no separate ESLint) |
| Grafana plugin / QAN app (`dashboards/pmm-app`) | `cd dashboards/pmm-app && yarn lint:check` (and `yarn typecheck` if TypeScript changed) |
| Before any PR | Run the row(s) that match **every** area you touched; fix errors, not just warnings, unless CI allows them |

---

## Change impact recipes

Recurring tasks — follow in order before opening a PR.

### Adding a REST API endpoint

1. Edit `api/<domain>/v1/*.proto` (HTTP annotations, validation rules).
2. Run `make gen`.
3. Implement handler/service logic in `managed/services/<domain>/`.
4. Add or extend tests in `api-tests/<domain>/`.
5. If UI-facing: add API module in `ui/apps/pmm/src/api/` and TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`.
6. If public API docs change: update [`docs/api/`](docs/api/) (PR template checkbox).

### Adding a DB table or migration

1. Add a versioned migration in `managed/models/database.go`.
2. Add or update the reform model; run `//go:generate` or `make gen`.
3. Add CRUD helpers in `*_helpers.go` or `*_crud.go` as surrounding code does.
4. Prefer `go-sqlmock` for unit tests; use `testdb.Open` when SQL/migration behavior must be verified.

### Adding a UI page or settings section

1. Read [`ui/AGENTS.md`](ui/AGENTS.md).
2. Add route in `ui/apps/pmm/src/router.tsx` if needed.
3. API functions in `ui/apps/pmm/src/api/`; TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`.
4. Co-locate Vitest tests (`*.test.ts` / `*.test.tsx`).
5. Run `cd ui && make lint && make test` before opening a PR.
6. JSON on the wire is **snake_case** (`axios-case-converter`); TypeScript uses **camelCase**.

---

## Git and pull request checklist

Full rules: [`docs/process/GIT_AND_GITHUB.md`](docs/process/GIT_AND_GITHUB.md). For commits and PR titles, use **[Conventional Commits](https://www.conventionalcommits.org/)** (`type(scope): summary`) — not the `PMM-XXXX` title style from the process doc. When opening PRs to the upstream Percona repo, confirm with reviewers if they expect conventional titles or `PMM-XXXX` titles from the process doc.

| Item | Rule |
|------|------|
| Branch name | `PMM-1234-short-description` (lowercase, dashes) |
| Commit title | `type(scope): short imperative summary` — e.g. `feat(ui): add OTEL settings scroll`, `fix(managed): normalize log parser YAML` |
| PR title | Same format as commit title (squash merge uses the PR title) |
| Types | `feat` (feature), `fix` (bug), `chore` (deps, lint, tooling), `refactor`, `test`, `docs` |
| Scope | Optional but preferred: `ui`, `managed`, `api`, `agent`, `adre`, `investigations`, `dashboards`, … |
| Ticket | Put `PMM-XXXX` in the branch name and/or PR body — not required in the title |
| Review fixes | New commit per round — do not amend and force-push |
| Merge | Squash and merge on GitHub |
| PR body | What/why, Feature Build link for features/fixes/improvements, link related PRs |
| API changes | Check API docs updated if endpoints changed |
| Before review | Tests and linters pass for every area touched (see [Linting decision tree](#linting-decision-tree); Go/API: `make prepare-pr`; UI: `cd ui && make lint`) |

---

## Feature areas

Some areas span multiple directories. When working on them, read **both** the component guide and the paths below. These paths exist on branches that land the work (e.g. ADRE, OTEL); if a directory is missing on your branch, skip it.

| Area | Backend | UI | Notes |
|------|---------|-----|-------|
| **ADRE / AI Assistant** | `managed/services/adre/` | `ui/apps/pmm/src/pages/adre/`, `components/adre/` | HolmesGPT integration, chat, usage |
| **Investigations** | `managed/services/investigations/` | `ui/apps/pmm/src/pages/investigations/` | AI investigation workflows |
| **Native QAN** | `qan-api2/` (existing `/v1/qan/*`) | `ui/apps/pmm/src/pages/qan/` | Native Query Analytics UI at `/pmm-ui/qan` (flag: `nativeQanEnabled`) |
| **OTEL** | `managed/otel/`, `dev/otel/` | Settings → OTEL tab | Log collectors, parser presets |
| **User docs** | — | — | [`documentation/`](documentation/) (MkDocs), not [`docs/process/`](docs/process/) |

When these areas grow large enough to need their own conventions, extend [`managed/AGENTS.md`](managed/AGENTS.md) and [`ui/AGENTS.md`](ui/AGENTS.md) — do not duplicate architecture here.

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

Core components and per-area guides: see [Component Guides](#component-guides) above.

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
| **Packer** | Machine image builds (AMI) |

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
| `make check` | Run Go/API linters (buf, golangci-lint, go-sumtype) |
| `make format` | Format code (gofumpt, goimports, gci) |
| `make release` | Build all binaries (agent, admin, managed, qan-api2) |
| `make test-common` | Run common unit tests |
| `make api-test` | Run API integration tests |
| `make prepare-pr` | Go/API pre-PR pipeline: `gen` + `check-all` (license + linters) + `format` + `go mod tidy` |
| `cd ui && make lint` | ESLint for PMM UI (required for UI changes; not part of `prepare-pr`) |

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

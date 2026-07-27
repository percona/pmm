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
| Human contributors | [`CONTRIBUTING.md`](CONTRIBUTING.md), [`dev/docs/process/`](dev/docs/process/) |
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
- Don't amend/squash commits locally to address review feedback; push **new commits** ([`dev/docs/process/GIT_AND_GITHUB.md`](dev/docs/process/GIT_AND_GITHUB.md)).
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
6. If public API docs change: update [`documentation/api/`](documentation/api/) (PR template checkbox).

> Keep proto changes additive — `buf breaking` runs in CI (see [Backward compatibility](#backward-compatibility)).

### Adding a DB table or migration

1. Add a versioned migration in `managed/models/database.go`.
2. Add or update the reform model; run `//go:generate` or `make gen`.
3. Add CRUD helpers in `*_helpers.go` or `*_crud.go` as surrounding code does.
4. Prefer `go-sqlmock` for unit tests; use `testdb.Open` when SQL/migration behavior must be verified.

> Migrations are forward-only — never edit or reorder one that has shipped (see [Backward compatibility](#backward-compatibility)).

### Adding a UI page or settings section

1. Read [`ui/AGENTS.md`](ui/AGENTS.md).
2. Add route in `ui/apps/pmm/src/router.tsx` if needed.
3. API functions in `ui/apps/pmm/src/api/`; TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`.
4. Co-locate Vitest tests (`*.test.ts` / `*.test.tsx`).
5. Run `cd ui && make lint && make test` before opening a PR.
6. JSON on the wire is **snake_case** (`axios-case-converter`); TypeScript uses **camelCase**.

---

## Running and verifying locally

Don't stop at unit tests — run PMM and exercise the change when it is user-visible.

1. `make env-up` starts the dev stack (`docker-compose.dev.yml`, waits for health).
2. Open **https://localhost** (accept the self-signed cert) and log in with PMM's default **`admin` / `admin`** (you're prompted to change it).
3. Inside the devcontainer, `make run-ui` serves the main UI with Vite HMR (port `5173`); `make run-qan-ui` serves the QAN plugin.
4. Call the REST API directly — paths live in `api/swagger/swagger.json` (e.g. `curl -k -u admin:admin https://localhost/v1/...`).
5. Handy ports: PostgreSQL `5432`, VictoriaMetrics `9090`, ClickHouse `9000`/`8123`, Delve `2345`, Mailhog UI `8025`.
6. Daemons log to stderr via the supervisor — read the logs for the component you changed.

---

## Definition of Done

Before calling a change complete, verify every item that applies — never report done on a check you didn't run:

- [ ] Builds (component build or `make release`) and, when you can exercise it, the app runs.
- [ ] Tests pass for **every** area you touched ([Testing decision tree](#testing-decision-tree)); new behavior has new/updated tests.
- [ ] The **matching** linter is clean ([Linting decision tree](#linting-decision-tree)); Go/API: `make prepare-pr`.
- [ ] Ran `make gen` if you changed `.proto`, reform models, or mocked interfaces — and did **not** hand-edit generated files.
- [ ] New source files carry the license header (`make check-license`); no secrets, credentials, or stray `fmt.Printf`/debug prints in the diff.
- [ ] Commits use `PMM-XXXX Short summary` and are signed off (`git commit -s`).
- [ ] Updated `AGENTS.md`/component guide **only** if structure, conventions, or workflows changed.

If a change is architecturally significant or spans multiple components, propose a short plan before mass-editing. Never delete or weaken tests just to make them pass, and don't invent APIs or fields — check the proto/generated code.

---

## Git and pull request checklist

Full rules: [`dev/docs/process/GIT_AND_GITHUB.md`](dev/docs/process/GIT_AND_GITHUB.md). PMM uses its own convention — **not Conventional Commits**. Commit and PR titles are `PMM-XXXX Short summary` (Jira key prefix, summary ≤50 chars, final period optional); do **not** use `type(scope):` prefixes.

| Item | Rule |
|------|------|
| Branch name | `PMM-1234-short-description` — start with `PMM-XXXX` (or `SAAS-XXXX`); lowercase, dashes, always a short description |
| Commit title | `PMM-XXXX Short summary` — Jira key prefix, ≤50 chars, imperative; final period optional. No `type(scope):` |
| Commit body | Blank line after the title, then an optional description wrapped at 72 chars |
| PR title | Same `PMM-XXXX Short summary` format (squash merge uses the PR title) |
| Ticket | `PMM-XXXX` is required as the title prefix (and in the branch name) |
| Sign-off | Sign commits with `git commit -s` (adds a `Signed-off-by` trailer — DCO convention; most PMM commits carry it) |
| Review fixes | New commit per round — do not amend and force-push |
| Merge | Squash and merge on GitHub |
| PR body | What/why, Feature Build link for features/fixes/improvements, link related PRs |
| API changes | Check API docs updated if endpoints changed |
| Before review | Tests and linters pass for every area touched (see [Linting decision tree](#linting-decision-tree); Go/API: `make prepare-pr`; UI: `cd ui && make lint`) |

---

## User documentation

User-facing docs are Markdown under [`documentation/docs/`](documentation/docs/). How to write them: [`docs-contributing.md`](documentation/docs-contributing.md) (workflow + local preview) and [`WRITERS-NOTES.md`](documentation/WRITERS-NOTES.md) (style, admonitions, variables, icons). MkDocs config lives in [`documentation/`](documentation/); this is separate from the developer process docs in [`dev/docs/process/`](dev/docs/process/).

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

Full schema, diagrams, and field-level detail: [`dev/docs/managed/data-model.md`](dev/docs/managed/data-model.md). Access-control (RBAC) architecture: [`dev/docs/managed/access-control.md`](dev/docs/managed/access-control.md).

## Repository Map

Core components and per-area guides: see [Component Guides](#component-guides) above.

### Supporting Directories

| Directory | Purpose |
|-----------|---------|
| `/dev/docs` | Developer docs: process (git workflow, tech stack, best practices) and managed architecture (data model, access control); public API docs live in `documentation/api/` |
| `/documentation` | User-facing documentation (MkDocs project root); pages live in `documentation/docs/` |
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

### Security and secrets
- Never log, hardcode, or commit secrets — credentials, tokens, S3 keys, TLS material. `.env`, `encryption.key`, and key material are gitignored; keep it that way.
- Persist sensitive values encrypted at rest via `managed/services/encryption` (rotation: `managed/cmd/pmm-encryption-rotation`) — not as plaintext columns.
- Enforce authorization in the service layer and respect RBAC ([`dev/docs/managed/access-control.md`](dev/docs/managed/access-control.md)); don't rely on the UI to hide privileged actions.
- Validate and bound external input; rely on reform's parameterized queries — never string-concatenate SQL.
- Report vulnerabilities per [`SECURITY.md`](SECURITY.md); keep exploit detail out of public issues and PRs.

### Concurrency and context propagation
- Thread `context.Context` as the first argument through call chains; honor cancellation and deadlines.
- Give every goroutine a clear exit tied to a context or `errgroup` — don't leak goroutines on shutdown (see [Graceful Shutdown](#graceful-shutdown)).
- Protect shared state with mutexes or channels; run `go test -race` on concurrency-sensitive packages.

### Backward compatibility
PMM Server talks to pmm-agents and API clients already deployed in the field, so changes must not break them.
- **Proto/API:** additive only — never renumber, retype, or remove an existing field. CI runs `buf breaking` against `api/descriptor.bin` (`cd api && make`).
- **DB migrations:** forward-only — add a new versioned migration in `managed/models/database.go`; never edit or reorder one that has shipped.
- Gate genuinely new behavior behind version checks (`version/features.go`) when older agents/servers must keep working.

### Dependencies and new files
- Prefer the standard library and deps already in `go.mod` / `ui/package.json`; a new dependency needs justification and an AGPL-3-compatible license (CI runs a license check).
- Go: `go get` then `go mod tidy` (both in `make prepare-pr`). UI: `yarn add` from `ui/`.
- New Go source files need the AGPL-3 Percona license header — copy it from an existing `.go` file or run `go tool license-eye -c .licenserc.yaml header fix`. Enforced by `make check-license` (exemptions: `agent/`, `admin/`, `utils/`, mocks).

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
| `make run-ui` | Inside devcontainer: Vite HMR for the main PMM UI |
| `make run-qan-ui` | Inside devcontainer: webpack + livereload for the QAN Grafana plugin |
| `make doc-build-preview` | Preview user docs (`documentation/docs/`) with live reload at http://localhost:8000 |
| `make doc-build` | Build user docs (used in CI); `make doc-build-pdf` for the PDF |
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
- `docker-compose.dev.yml` — development environment (PMM Server, renderer)
- `docker-compose.yml` — community/quickstart compose (stable image, minimal config)
- `go.mod` — Go module definition
- `.golangci.yml` — linter configuration
- `.mockery.yaml` — mock generation configuration
- `dev/docs/process/tech_stack.md` — technology choices and rationale
- `dev/docs/process/best_practices.md` — coding best practices
- `dev/docs/process/GIT_AND_GITHUB.md` — git workflow
- `dev/docs/process/v2_to_v3_environment_variables.md` — v2→v3 environment variable migration
- `dev/docs/managed/data-model.md` — inventory data model (schema + diagrams)
- `dev/docs/managed/access-control.md` — access control (RBAC) architecture

# PMM Development Guide for AI Agents

## How agents load this file

`AGENTS.md` is the tool-neutral source of truth. Read it and the component guides directly, whatever the assistant.

Pointer files that route here (examples): Claude Code `CLAUDE.md`, Gemini CLI `GEMINI.md`, Copilot `.github/copilot-instructions.md`. Cursor loads root and nested `AGENTS.md` natively.

Claude Code reads `CLAUDE.md`, not `AGENTS.md`, so every directory with a component guide has a `CLAUDE.md` containing only `@AGENTS.md`. It loads on demand when a file in that directory is first read. Keep pointers to the import alone; guidance belongs in `AGENTS.md`.

Personal AI-tool files (`.claude/`, `.cursor/`) are gitignored, except `.claude/settings.json` (team config).

If `AGENTS.local.md` (repo root) or `~/AGENTS.local.md` exists, read it at session start: machine-specific paths, gitignored. Never put secrets there.

## Maintaining this file

You keep this file accurate. After work, update it (and the matching component guide) if you:

- added, removed or renamed a top-level directory or component
- added or removed a component `AGENTS.md` (add or remove its `CLAUDE.md` pointer too)
- changed the tech stack (`go.mod` dependency, tool added or removed)
- changed build targets in `Makefile` / `Makefile.include`
- changed global conventions (style, error handling, testing)
- changed architecture or data flow (pipeline, protocol)
- changed the dev environment (`docker-compose.yml`, `.devcontainer/`)

Do not update it for routine bug fixes or minor features.

## How docs are organized

Humans: `CONTRIBUTING.md`, `dev/docs/process/`. Agents: this file plus component guides. This file summarizes and links process docs; it repeats a rule only when agents routinely get it wrong.

### Component Guides

Read the matching guide before working on a component:

- pmm-managed (server backend): `managed/AGENTS.md`
- pmm-agent (client agent): `agent/AGENTS.md`
- pmm-admin (CLI): `admin/AGENTS.md`
- APIs (protobuf): `api/AGENTS.md`
- qan-api2 (query analytics): `qan-api2/AGENTS.md`
- vmproxy (VictoriaMetrics proxy): `vmproxy/AGENTS.md`
- UI (React): `ui/AGENTS.md`
- Dashboards (Grafana JSON): `dashboards/dashboards/AGENTS.md`
- QAN App (Grafana plugin, QAN panel): `dashboards/pmm-app/AGENTS.md`
- API tests (integration): `api-tests/AGENTS.md`
- Build and packaging: `build/AGENTS.md`
- Documentation (user docs, release notes): `documentation/AGENTS.md`

A component guide covers only its area. Global conventions (Go style, errors, logging, testing, codegen) live in Global Development Conventions below and are not repeated in component guides: both load together, so a restated rule costs context twice and drifts. Put a new rule in the most specific guide(s) that cover it, nowhere else. The one exception is "PMM-specific choices", which repeats a few pitfalls on purpose.

---

## How AI agents should work in this repo

The `AGENTS.md` hierarchy is the single source of truth. Follow every section through "Git and pull request checklist" on every code change; read "Product Overview" onward for context.

### Workflow

1. Identify the component (`managed`, `ui`, `api`, `agent`, ...).
2. Read its `AGENTS.md` before planning or editing.
3. Keep diffs minimal and focused; match surrounding style.
4. After `.proto` or reform model changes: `make gen` from repo root.
5. Run the smallest covering test set (Testing decision tree).
6. Run the matching linter before calling work PR-ready (Linting decision tree). For Go/API, step 7 covers it.
7. Go/API: `make prepare-pr` (gen + license check + Go lint + format + `go mod tidy`).
8. Update `AGENTS.md` / component guide only if structure, conventions or workflows changed.

User-visible changes (metric, dashboard, API, exporter, UI) need verification on a live PMM server with real data, not just unit tests. See Definition of Done and `dev/docs/process/running-and-verifying-locally.md`.

### Don'ts

- Don't edit generated files (`.pb.go`, `.pb.gw.go`, `*_reform.go`, `*.pb.validate.go`, swagger specs, `json/client/`).
- Don't use `gorm` in pmm-managed; reform only.
- Don't amend/squash to address review feedback; push new commits (`dev/docs/process/GIT_AND_GITHUB.md`).
- Don't force-push to `main`/`v3`.
- Don't skip the Feature Build link in PR descriptions for user-facing changes (`.github/pull_request_template.md`).
- Don't run the full linter on every tiny edit; do run the targeted linter, and `make prepare-pr` before calling Go/API work PR-ready.
- Don't write unit tests that call external services; use mocks or `/api-tests/`.

---

## PMM-specific choices (agents often get wrong)

These differ from generic Go/React advice. Match surrounding code; when in doubt, follow the component guide.

- DB (managed): reform only, never gorm or other ORMs (`managed/AGENTS.md`)
- Unit tests (managed): `go-sqlmock` by default; `testdb.Open` only when migrations or fixtures are under test (`managed/AGENTS.md`)
- API errors (Go): `status.Error()` with gRPC codes, not ad-hoc HTTP errors in service layers
- Logging (Go): `logrus` with `*logrus.Entry` and structured fields, not `fmt.Printf`
- Mocks (Go): small interfaces in `deps.go` + mockery, not hand-rolled fakes
- UI server state: TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`, not `useEffect` + `fetch` (`ui/AGENTS.md`)
- UI client state: React Context for auth/settings, not Redux or another global store
- UI components: MUI + `@percona/peak-ui`, theme-aware `sx`, not ad-hoc CSS
- UI wire format: camelCase in TypeScript, snake_case JSON on the wire (`axios-case-converter` in `ui/apps/pmm/src/api/api.ts`)
- Generated code: edit `.proto` / reform models / interfaces, run `make gen`; never hand-edit `*.pb.go`, `*_reform.go`, swagger clients

Mechanical style is enforced by `make check` (Go), `cd ui && make lint && make format-check` (UI), and CI.

---

## Testing decision tree

Three layers (`CONTRIBUTING.md`): unit, API integration, e2e (in percona/pmm-qa). Use the smallest scope:

- Go logic in one package: `go test ./path/to/pkg/...` or `make test` in the component dir
- Shared/API packages (not managed/admin/agent): `make test-common` from root
- `managed/models`, DB schema or migrations: unit tests in `managed/`; `testdb.Open` only when fixtures or migrations matter
- `.proto` or gRPC/REST definitions: `make gen`, then `make check`; update `managed/` handlers and UI hooks if user-facing
- REST end-to-end: `make env-up`, then `make api-test` (`api-tests/AGENTS.md`)
- UI (`ui/`): `cd ui && make lint && make test`
- Dashboard JSON (`dashboards/dashboards/`): `python3 dashboards/misc/cleanup-dash.py --check-only <file>` (or without `--check-only`); CI enforces it in `dashboards.yml`
- User docs (`documentation/`): `make doc-build-preview` and read the rendered page; CI runs `linkspector` on docs PRs

## Linting decision tree

CI runs separate linters per area. `make prepare-pr` covers Go only, not UI or dashboards.

The Go linter is `bin/golangci-lint`, pinned to CI's version. Install it with `make init` only; another build reports different findings.

- Go backend (`managed/`, `agent/`, `admin/`, `qan-api2/`, `vmproxy/`, shared packages): `make prepare-pr` from root (or `make check` after `make gen` for a quick pass)
- `.proto` only: `make gen`, then `make check` (`buf lint`, `golangci-lint`, `go-sumtype`)
- UI (`ui/`): `cd ui && make lint && make format-check` (oxlint + oxfmt over all workspace packages; same as CI `ui.yml`)
- Dashboard JSON: `python3 dashboards/misc/cleanup-dash.py --check-only <file>` before commit (CI `dashboards.yml`)
- `dashboards/misc/cleanup-dash.py`, `dashboards/misc/test_*.py`: no linter; run the suite (Testing decision tree). Other scripts in `dashboards/misc/` have no automated coverage
- Grafana plugin / QAN app (`dashboards/pmm-app`): `yarn lint:check` there (plus `yarn typecheck` if TypeScript changed)
- Before any PR: run the entry for every area touched; fix errors, not just warnings, unless CI allows them

---

## Change impact recipes

### Adding a REST API endpoint

1. Edit `api/<domain>/v1/*.proto` (HTTP annotations, validation).
2. `make gen`.
3. Implement logic in `managed/services/<domain>/`.
4. Add tests in `api-tests/<domain>/`.
5. If UI-facing: API module in `ui/apps/pmm/src/api/`, TanStack Query hooks in `ui/apps/pmm/src/hooks/api/`.
6. If public API docs change: update `documentation/api/` (PR template checkbox).

Keep proto changes additive; `buf breaking` runs in CI.

### Adding a DB table or migration

1. Add a versioned migration in `managed/models/database.go`.
2. Add/update the reform model; `//go:generate` or `make gen`.
3. Add CRUD helpers in `*_helpers.go` or `*_crud.go` like surrounding code.
4. Prefer `go-sqlmock`; `testdb.Open` when SQL/migration behavior must be verified.

Migrations are forward-only; never edit or reorder a shipped one.

### Adding a UI page or settings section

1. Read `ui/AGENTS.md`.
2. Add route in `ui/apps/pmm/src/router.tsx` if needed.
3. API functions in `ui/apps/pmm/src/api/`; hooks in `ui/apps/pmm/src/hooks/api/`.
4. Co-locate Vitest tests (`*.test.ts(x)`).
5. `cd ui && make lint && make test` before the PR.
6. Wire JSON is snake_case, TypeScript is camelCase.

---

## Definition of Done

Verify every item that applies; never report done on a check you didn't run:

- [ ] Builds (component build or `make release`); the app runs when you can exercise it.
- [ ] Tests pass for every area touched; new behavior has new/updated tests.
- [ ] Matching linter clean; Go/API: `make prepare-pr`.
- [ ] Ran `make gen` after `.proto`, reform model or mocked-interface changes; no hand-edited generated files.
- [ ] New source files have the license header (`make check-license`); no secrets or stray debug prints.
- [ ] Commits are `PMM-XXXX Short summary` and signed off (`git commit -s`).
- [ ] `AGENTS.md`/component guide updated only if structure, conventions or workflows changed.

Propose a short plan before mass-editing architecturally significant or cross-component changes. Never delete or weaken tests to make them pass. Don't invent APIs or fields; check the proto/generated code.

User-visible changes: run PMM on a live server, reproduce, deploy the fix, and verify with evidence (dashboards, metrics/API, logs) across the supported version matrix. Procedure: `dev/docs/process/running-and-verifying-locally.md`.

---

## Git and pull request checklist

Full rules: `dev/docs/process/GIT_AND_GITHUB.md`. PMM does not use Conventional Commits; no `type(scope):` prefixes.

- Branch: `PMM-1234-short-description` (or `SAAS-XXXX`); lowercase, dashes, always a description
- Commit title: `PMM-XXXX Short summary`, ≤50 chars, imperative, final period optional
- Commit body: blank line after the title, optional description wrapped at 72 chars
- PR title: same format (squash merge uses it)
- Ticket: `PMM-XXXX` required as title prefix and in the branch name
- Sign-off: `git commit -s` (DCO trailer)
- Review fixes: new commit per round; no amend + force-push
- Merge: squash and merge on GitHub
- PR body: what/why, Feature Build link for features/fixes/improvements, related PRs
- API changes: API docs updated if endpoints changed
- Before review: tests and linters pass for every area touched (Go/API: `make prepare-pr`; UI: `cd ui && make lint`)

### Handling review comments (incl. bots like CodeRabbit)

- Bot findings are claims to verify, not facts. CodeRabbit mixes real catches with false positives. Fix real ones (say what you verified), skip the rest with a brief reason, keep changes minimal.
- With `gh`: `gh api --paginate repos/percona/pmm/pulls/<N>/comments` (inline), `.../issues/<N>/comments` (top-level), or `gh pr view <N> --comments`.
- Without `gh` (web/sandboxes, 60 req/hour unauthenticated, don't loop): `curl -s 'https://api.github.com/repos/percona/pmm/pulls/<N>/comments?per_page=100'` for up to 100 inline comments; top-level at `issues/<N>/comments`; full thread via `WebFetch` on the PR URL.

---

## User documentation

User docs are Markdown in `documentation/docs/`. Read `documentation/AGENTS.md` before editing anything under `documentation/` (voice, structure, Markdown, release notes); it links `documentation/WRITERS-NOTES.md` (admonitions, icons, symbols) and `documentation/CONTRIBUTING.md` (workflow, local preview). MkDocs config is in `documentation/`, separate from `dev/docs/process/`.

A merge to `main` publishes docs live; don't merge docs for an unshipped feature.

---

## Product Overview

Percona Monitoring and Management (PMM) is open-source monitoring for MySQL, MongoDB, PostgreSQL, ProxySQL, HAProxy, Valkey and cloud databases (AWS RDS, Azure). Client-server: lightweight agents on monitored hosts send metrics and query analytics to a central server for storage, alerting and visualization.

This monorepo holds PMM components, APIs, docs and build scripts. Backend is Go; UI is TypeScript/React.

## Architecture and Data Flow

Metrics: exporters (node, mysqld, mongodb, postgres, proxysql, valkey, rds, azure) → VMAgent (scrapes) → VictoriaMetrics (on PMM Server) → Grafana; and VictoriaMetrics → VMAlert → Alertmanager.

QAN: QAN agents in pmm-agent (perfschema, slowlog, pg_stat_statements, pg_stat_monitor, MongoDB profiler/log) → pmm-managed (gRPC) → qan-api2 (gRPC) → ClickHouse → PMM UI / Grafana.

Agent communication: pmm-agent ↔ pmm-managed over a bidirectional gRPC stream. Server sends SetStateRequest, StartAction, StartJob, Ping. Agent sends StateChanged, QanCollect, ActionResult, JobResult, Pong.

Backup: pmm-managed orchestrates → pmm-agent jobs (PBM for MongoDB, mysqldump/xtrabackup for MySQL) → S3/MinIO/local storage.

## Domain Model

Inventory is Node → Service → Agent:

- Node: physical or virtual host (generic, container, remote, RDS, Azure)
- Service: database or app on a node (MySQL, MongoDB, PostgreSQL, ProxySQL, HAProxy, Valkey, external)
- Agent: monitoring agent on a node, optionally for a service (pmm-agent, exporters, QAN agents, VMAgent)

A Node has many Services; a Service belongs to one Node. An Agent runs on a Node (`runs_on_node_id`) and optionally monitors a Service (`service_id`). A child Agent belongs to a parent PMM Agent (`pmm_agent_id`).

Schema and diagrams: `dev/docs/managed/data-model.md`. RBAC: `dev/docs/managed/access-control.md`.

## Repository Map

Core components: see Component Guides. Supporting directories:

- `/dev/docs`: developer docs (process, managed architecture); public API docs are in `documentation/api/`
- `/documentation`: user docs (MkDocs root); pages in `documentation/docs/`
- `/version`: version info and feature flags
- `/dev`: dev utilities (e.g. mongo-rs-backups)
- `/.devcontainer`: devcontainer setup

External repos (github.com/...):

- percona/grafana: Grafana fork with PMM customizations
- percona/node_exporter, mysqld_exporter, mongodb_exporter, postgres_exporter, proxysql_exporter, rds_exporter, azure_metrics_exporter: exporters
- percona/pmm-qa: e2e UI tests, QA DB setups, CLI tests
- Percona-Lab/pmm-submodules: Feature Build orchestration

## Tech Stack

- Go: all backend components
- TypeScript/React: PMM UI (`/ui`)
- Protobuf v3 / gRPC: APIs and inter-component communication; grpc-gateway generates the REST API
- PostgreSQL: pmm-managed store (inventory, settings, backups)
- ClickHouse: QAN store (qan-api2)
- VictoriaMetrics: time series; VMAlert: alert rules; Grafana: dashboards
- reform: Go ORM, NOT gorm. pmm-managed's PostgreSQL store and pmm-agent's row mappers for monitored MySQL/PostgreSQL system views
- logrus: structured logging
- testify: `assert` and `require` only, NOT suites
- mockery: mock generation
- golangci-lint: static analysis
- Kong: pmm-admin CLI framework
- Docker Compose: dev environment
- Ansible: server provisioning; Packer: AMI builds

## Global Development Conventions

### Code Style
- Format with `gofumpt -s`; run `make format`
- Import groups: stdlib, external (`github.com/percona`, third-party), internal
- `any` instead of `interface{}`
- Modern slice helpers (`slices.Contains`), range loops
- `sync.WaitGroup.Go` instead of `Add`/`go func`/`Done`; don't copy loop variables for closures (Go 1.22+ per-iteration scoping)
- No named return values
- No inline comments (`code // comment`) except `//nolint`
- No inline `err != nil` checks (`if err := f(); err != nil`); assign, then check on the next line
- No obvious comments; comment only non-obvious intent

### Error Handling
- `status.Error()` with proper gRPC codes for API errors
- Wrap with context: `fmt.Errorf("descriptive context: %w", err)`
- Return early on errors
- Inspect with `errors.Is()`, `errors.As()` or `errors.AsType()`
- Standard `errors`, not `github.com/pkg/errors` (existing uses may remain)
- No `%q` in error messages; use `%s`, or `'%s'` when the value can contain spaces

### Logging
- `logrus` with structured fields
- Pass `*logrus.Entry`, not `*logrus.Logger`
- Format: `s.l.WithField("key", value).Error("message")`
- No `%q` in log messages; use `%s`, or `'%s'` when the value can contain spaces
- Log to unbuffered stderr; the supervisor handles the rest

### Environment Variables
- `PMM_DEV_*`: dev/test only, never for end users
- `PMM_TEST_*`: not GA
- `PMM_*`: GA
- Sub-prefixes for component groups (e.g. `PMM_HA_*`)

### Testing
- `testify/assert` and `testify/require`, not suites
- `t.Context()` over `context.Background()`; `usetesting` misses it inside helper closures
- Mocks via `mockery` (`.mockery.yaml`)
- Unit tests: `*_test.go` beside the code
- Integration tests: `/api-tests/`, against a live PMM Server
- E2E: percona/pmm-qa

### Code Generation
- Protobuf/gRPC: `make gen` from root
- Mocks: `mockery` per `.mockery.yaml`
- Never edit generated files (`.pb.go`, `.pb.gw.go`, `*_reform.go`, `*.pb.validate.go`, swagger specs, `json/client/`)

### Security and secrets
- Never log, hardcode or commit secrets (credentials, tokens, S3 keys, TLS material). Keep `.env`, `encryption.key` and key material gitignored.
- Persist sensitive values encrypted via `managed/services/encryption` (rotation: `managed/cmd/pmm-encryption-rotation`), not plaintext columns.
- Enforce authorization in the service layer and respect RBAC (`dev/docs/managed/access-control.md`); don't rely on the UI hiding actions.
- Validate and bound external input; use reform's parameterized queries, never concatenated SQL.
- Report vulnerabilities per `SECURITY.md`; keep exploit detail out of public issues and PRs.

### Concurrency and context propagation
- `context.Context` is the first argument through call chains; honor cancellation and deadlines.
- Every goroutine exits via a context or `errgroup`; no leaks on shutdown (Graceful Shutdown).
- Protect shared state with mutexes or channels; `go test -race` on concurrency-sensitive packages.

### Backward compatibility
PMM Server talks to deployed pmm-agents and API clients; don't break them.
- Proto/API: additive only; never renumber, retype or remove a field. CI runs `buf breaking` against `api/descriptor.bin` (`cd api && make`).
- DB migrations: forward-only; add a new versioned migration in `managed/models/database.go`, never edit or reorder a shipped one.
- Gate new behavior behind version checks (`version/features.go`) when older agents/servers must keep working.

### Dependencies and new files
- Prefer stdlib and existing deps (`go.mod`, `ui/package.json`); a new dependency needs justification and an AGPL-3-compatible license (CI checks).
- Go: `go get`, then `go mod tidy` (both in `make prepare-pr`). UI: `pnpm add` in `ui/` (or `pnpm --filter <pkg> add`).
- New Go files need the AGPL-3 Percona header: copy from an existing `.go` file or `go tool license-eye -c .licenserc.yaml header fix`. Enforced by `make check-license` (exempt: `agent/`, `admin/`, `utils/`, mocks).

### Graceful Shutdown
- `SIGTERM`/`SIGINT` cancel the parent context
- Stop handling signals after the first, so a second one terminates immediately
- Startup errors are fatal; runtime errors are handled, logged and communicated

### Debug Endpoints
Long-running daemons expose on `127.0.0.1`: `/debug/metrics` (Prometheus), `/debug/vars` (expvar), `/debug/requests` and `/debug/events` (tracing), `/debug/pprof`.

## Key Make Targets

- `make init`: install pinned dev tools into `bin/` (CI's golangci-lint)
- `make env-up`: start the dev container (PMM Server); `make env-up-rebuild` rebuilds from scratch
- `make env TARGET=<t>`: run `make <t>` inside `pmm-server` as `pmm` (bash if no `TARGET`); `make env-root` for build/test/lint
- `make env-root TARGET=run-managed-ci`: rebuild and hot-swap pmm-managed without an image rebuild (see `dev/docs/process/running-and-verifying-locally.md`); also `run-agent-ci`, `run-qan-ci`, `run-vmproxy-ci`, `run-all`
- `make run-ui`: in devcontainer, Vite HMR for the PMM UI
- `make run-qan-ui`: in devcontainer, webpack + livereload for the QAN plugin
- `make doc-build-preview`: preview user docs at http://localhost:8000; `make doc-build` (CI), `make doc-build-pdf`
- `make gen`: all codegen (protobuf, reform, mocks, format)
- `make check`: Go/API linters (buf, golangci-lint, go-sumtype)
- `make format`: gofumpt, gci
- `make release`: build agent, admin, managed, qan-api2
- `make test-common`: common unit tests
- `make api-test`: API integration tests
- `make prepare-pr`: `gen` + `check-all` (license + linters) + `format` + `go mod tidy`
- `cd ui && make lint`: oxlint (required for UI; not in `prepare-pr`)
- `cd ui && make format-check`: oxfmt check (CI); `make format` writes

## Key Files to Reference

- `Makefile`, `Makefile.include`: build and dev targets
- `docker-compose.dev.yml`: dev environment (PMM Server, renderer)
- `docker-compose.yml`: quickstart compose (stable image)
- `go.mod`, `.golangci.yml`, `.mockery.yaml`
- `dev/docs/process/`: `tech_stack.md`, `best_practices.md`, `GIT_AND_GITHUB.md`, `running-and-verifying-locally.md` (run locally, register test DBs, evidence), `v2_to_v3_environment_variables.md`
- `dev/docs/managed/`: `data-model.md`, `access-control.md`

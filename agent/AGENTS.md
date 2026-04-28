# pmm-agent Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [managed/AGENTS.md](../managed/AGENTS.md) (server backend) · [api/AGENTS.md](../api/AGENTS.md) (API definitions) · [admin/AGENTS.md](../admin/AGENTS.md) (CLI)

**pmm-agent** is the client-side monitoring agent for PMM. It runs on every monitored host, managing metric exporters as child processes, running built-in QAN and Real-Time Analytics (RTA) collectors in-process, executing on-demand actions (explain, PT summary), and performing backup/restore jobs. It communicates with pmm-managed on the PMM Server over a persistent bidirectional gRPC stream.

## Architecture

### Core Design: Supervisor Pattern

The agent receives a **desired state** (`SetStateRequest`) from pmm-managed describing which agents should be running. The Supervisor computes a diff (toStart, toRestart, toStop) and reconciles actual state to match.

### Two Agent Categories

1. **Process agents (exporters)** — external binaries run as child processes via `agents/process/`. Each has a state machine: `STARTING → RUNNING` (or `FAILING` with backoff). Includes node_exporter, mysqld_exporter, mongodb_exporter, postgres_exporter, proxysql_exporter, rds_exporter, azure_exporter, valkey_exporter, VMAgent, Nomad.

2. **Built-in agents** — Go code implementing the `BuiltinAgent` interface, run in-process. Includes QAN collectors (perfschema, slowlog, pg_stat_statements, pg_stat_monitor, MongoDB profiler, mongolog) and RTA agents (MongoDB realtimeanalytics).

### Communication with PMM Server

The `client` package maintains a persistent bidirectional gRPC stream (`Agent.Connect`) with reconnect/backoff:

- **Server → Agent**: `SetStateRequest`, `StartAction`, `StopAction`, `CheckConnection`, `StartJob`, `StopJob`, `Ping`, `GetVersions`, `PBMSwitchPITR`, `AgentLogs`
- **Agent → Server**: `StateChanged`, `QanCollect`, `ActionResult`, `JobResult`, `Pong`
- A separate RTA channel streams `CollectRequest` data via client-streaming RPC

### Local API

`agentlocal` exposes a local gRPC + JSON API for status, Prometheus metrics, pprof debug endpoints, and config reload.

## Key Packages and Responsibilities

| Package | Responsibility |
|---------|---------------|
| `commands` | CLI commands: `run` (main event loop), `setup` (registration with server) |
| `config` | YAML + CLI flags + env vars configuration; thread-safe `Storage` for runtime access |
| `client` | gRPC client managing persistent connection to pmm-managed, message routing |
| `client/channel` | Bidirectional gRPC stream abstraction with request/response correlation |
| `agentlocal` | Local API server: status, reload, debug endpoints |
| `agents/supervisor` | Central lifecycle manager: start/stop/restart agents, port allocation |
| `agents/process` | External process wrapper: FSM (STARTING→RUNNING→FAILING), backoff, logging |
| `agents/mysql`, `postgres`, `mongodb` | Built-in QAN and RTA collectors |
| `runner/actions` | Short-lived actions: MySQL/PostgreSQL/MongoDB explain, show create table, PT summary |
| `runner/jobs` | Long-lived jobs: MySQL backup/restore (mysqldump, xtrabackup), MongoDB backup/restore (PBM) |
| `connectionchecker` | Verifies connectivity to MySQL, PostgreSQL, MongoDB, ProxySQL |
| `serviceinfobroker` | Discovers service metadata (versions, tables, databases) |
| `utils/templates` | Renders exporter args/env/config files from server-provided Go templates |

## Domain Model

pmm-agent has **no direct database access**. All state comes from pmm-managed via gRPC:

- `SetStateRequest` contains the desired set of agents with their configurations
- `AgentProcess` / `BuiltinAgent` are the runtime representations managed by the Supervisor
- Exporter configuration (args, env, text files) is templated from server-provided data using `listen_port`, `paths_base`, and other variables
- VMAgent receives a Prometheus scrape config rendered by pmm-managed

## Configuration

- **Sources**: YAML file (`pmm-agent.yaml`), CLI flags, environment variables (`PMM_AGENT_*`)
- **Runtime access**: `config.Storage` provides thread-safe `Get()` and `Reload()`
- **Key settings**:
  - `server.address`, `server.username`, `server.password` — PMM Server connection
  - `paths.exporters_base` — base directory for exporter binaries
  - `paths.tempdir` — temporary directory for rendered config files
  - `ports.min`, `ports.max` — port range for exporter listen addresses
  - Per-exporter paths: `paths.node_exporter`, `paths.mysqld_exporter`, etc.

## Patterns and Conventions

### Do
- Use `config.Storage` for thread-safe config access
- Implement `BuiltinAgent` interface for new in-process collectors (methods: Run, Changes, Describe, Collect)
- Use `process.Process` for new external exporters
- Use table-driven tests with golden files for parsers
- Use `utils/templates` to render exporter args from server-provided templates
- Follow the supervisor pattern — let the supervisor manage all agent lifecycle

### Don't
- Don't hardcode exporter binary paths — use `config.Paths`
- Don't bypass the supervisor for agent lifecycle management
- Don't use raw SQL — the agent has no database; all data comes via gRPC
- Don't modify exporter args directly — they come from server templates

## Testing

- **Unit tests**: `make test` (with race detector, sequential `-p 1`)
- **Integration environment**: `make env-up` starts docker-compose with MySQL, MongoDB, PostgreSQL, Valkey
- **Database shells**: `make env-mysql`, `make env-mongo`, `make env-psql`
- **Fuzz tests**: `make fuzz-slowlog-parser`, `make fuzz-postgres-parser`
- **Benchmarks**: `make bench` for slowlog and postgres parsers
- **Coverage**: `maincover_test.go` with `maincover` build tag
- **Golden files**: parser tests use golden files in `testdata/` directories
- **Linting**: `make check-all` before submitting PRs

## Code Generation

- **reform**: not used (agent has no DB)
- **mockery**: generates mocks for supervisor, connectionChecker, serviceInfoBroker interfaces
- **protobuf**: agent consumes types from `/api`; run `make gen` from repo root if proto files change

## Key Files to Reference

- `agent/main.go` — entry point, CLI setup
- `agent/commands/run.go` — main event loop wiring client, supervisor, runner
- `agent/agents/supervisor/supervisor.go` — central agent lifecycle manager
- `agent/agents/process/process.go` — external process state machine
- `agent/client/client.go` — gRPC client and message routing
- `agent/config/config.go` — configuration structure and loading
- `agent/docker-compose.yml` — integration test environment
- `agent/Makefile` — build, test, and development targets

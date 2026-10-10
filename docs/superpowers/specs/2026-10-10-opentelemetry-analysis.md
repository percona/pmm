# OpenTelemetry in PMM: analysis and design (PMM-15567)

- **Date:** 2026-10-10
- **Epic:** [PMM-15567](https://perconadev.atlassian.net/browse/PMM-15567), with 21 child tickets (listed in [§2](#2-sources)).
- **Prototype:** [percona/pmm#5118](https://github.com/percona/pmm/pull/5118), branch `tibi-holmes`. It is a reference only.
- **Code base:** `main` at `285cefaf8`. Line numbers below refer to that commit; re-check them before you edit, because `main` moves.
- **Plans that implement this document:** [`docs/superpowers/plans/2026-10-10-opentelemetry-00-roadmap.md`](../plans/2026-10-10-opentelemetry-00-roadmap.md) and plans 01 to 06 next to it.

Marks used below:
- **[V]** means verified by reading the code or the ticket.
- **[I]** means inferred, or taken from upstream documentation, and not run.

---

## 1. Summary

**What gets built.**
- An OpenTelemetry Collector, built with OCB from pinned source, runs in two places:
  - in PMM Server as a receiver: OTLP/HTTP on loopback → redaction → batch → ClickHouse database `otel`;
  - on every PMM Client node, as a new pmm-agent–managed agent type: file and journald log sources → OTLP/HTTP → `https://<server>/otlp/`.
- pmm-managed owns the node collector's config, log sources and parser presets.
- qan-api2 migrations own the ClickHouse schema.
- Grafana reads the data through a read-only ClickHouse datasource.
- The epic already records most of the design decisions (24 Sep 2026, ADR list in PMM-15590). This document builds on them and does not reopen them, except where the code on `main` makes a decision unworkable as written ([§5](#5-findings-on-main-that-change-the-tickets)).

**Where the requested goal and the epic differ.** The goal in the planning request and the epic don't match on five points:

| Goal | What the epic says [V] | Recommendation |
|---|---|---|
| The server receiver takes logs, traces, profiles and eBPF data | Logs in Phase 1a. Traces in Phase 2 (PMM-15579). eBPF moved to its own epic, PMM-15588. Profiles are not mentioned anywhere. OTLP metrics ingestion and export are out of scope (PMM-15391). | Keep the epic's phasing. Build the server config generator per signal, so traces become a table plus a pipeline entry in Phase 2. Don't build profiles now: the OTel profiles signal has been public Alpha since March 2026 [I], and the contrib ClickHouse exporter marks profiles as "development" stability [V, exporter README]. eBPF data reaches PMM as OTLP traces (ClickHouse, Phase 2) and as metrics (VictoriaMetrics, PMM-15588), so it needs no separate receiver here. |
| The client collector collects database logs, traces, profiles and eBPF | Logs in Phase 1. A traces receiver on the node in Phase 2. eBPF in PMM-15588. | Same as above. |
| `pmm-admin add mysql/postgresql/mongodb` collects the database log by default, from the default location or from the database | It is opt-in through `--collect-logs`, and paths come from the database (PMM-15584/85/86). PMM-15584 says "(default `error`)" but its tests pass the flag explicitly, so the ticket is ambiguous. | Collect the **error log by default** when the service is on the same host as pmm-agent. `--collect-logs=none` opts out. Slow and general logs stay opt-in, because they contain SQL text and personal data, and QAN already reads the slow log. Read paths **from the database only**; don't guess distro defaults ([§6.3](#63-default-log-collection-on-pmm-admin-add)). **Decided: D2.** |
| Parsers can be added in the PMM UI for any log file, and pmm-client uses them | Custom presets are managed through the API and Settings → OTEL (PMM-15574). The tickets don't say whether the UI can add or bind log sources. | Add a log-source editor to Settings → OTEL that uses the same API as `pmm-admin add logs`. The node's allow-list still checks every hand-typed path. A "test this preset on sample lines" preview is a follow-up, not part of this work ([§6.4](#64-parsers-in-the-ui)). **Decided: D7.** |
| How LBAC works with this | Admins only, until LBAC covers logs. | Phase 1: Admin-only, enforced in the Grafana fork; PMM-15576's approach can't work ([§5](#5-findings-on-main-that-change-the-tickets)). Later: a PMM logs query API that turns the user's LBAC selectors into an allow-list of service and node IDs and fails closed ([§7](#7-lbac)). |

**Decisions:** [§8](#8-decisions). D1–D17 were decided in the interview on 2026-10-10. D2, D11 and D12 change the epic or its tickets, so they go to the epic owner and the architects as Proposed ADRs (ADR-19, ADR-20, and an amendment to PMM-15574/15571) before coding.

---

## 2. Sources

- **Epic and children** [V], all read in full:
  - Phase 0: PMM-15590 (ADRs), PMM-15591 (threat model), PMM-15592 (capacity).
  - Phase 1a: PMM-15571 (server collector), PMM-15572 (agent type), PMM-15574 (presets), PMM-15575 (server's own logs), PMM-15576 (Grafana), PMM-15577 (packaging), PMM-15593 (self-metrics), PMM-15594 (HA), PMM-15578 (QA).
  - Phase 1b: PMM-15573 (`add logs`), PMM-15584/85/86 (discovery), PMM-15587 (database presets).
  - Phase 2: PMM-15579 (traces), PMM-15580 (traces dashboard), PMM-15582 (correlation).
  - Separate epic: PMM-15588 (eBPF).
  - Also returned as a child: PMM-4180 (PostgreSQL slowlog agent, On Hold). It is unrelated.
- **Prototype:** PR #5118. 491 files, +76k lines, 265 commits, far behind `main`. It mixes OTel with ADRE/Holmes AI work. Nobody has reviewed it: every review thread comes from the lint bot, and the codecov patch coverage is 2.88 %. [V]
- **Older prior art, also unmerged:** branch `PMM-14689-logs-and-traces-phase1` (last merge 2026-08-27) [V].
  - It ships `otelcol-contrib` 0.120.0 through an Ansible role.
  - pmm-managed owns the schema there, in its own migration chain (`logs_schema_migrations`), with tables `pmm.logs` and `pmm.traces` and retention through `MODIFY TTL`.
  - Client logs travel over a gRPC `LogShipService`.
  - The datasource uses the privileged ClickHouse user.
  - It is useful for its filelog configs and tests. Its architecture is superseded by the epic's decisions.
- **`main`:** pmm-managed, pmm-agent, pmm-admin, qan-api2, build/, ui/, plus percona/grafana and percona-helm-charts (pmm-ha chart 1.6.1, ClickHouse 25.3).

---

## 3. Target architecture

```
PMM Client node                                          PMM Server
───────────────                                          ──────────
pmm-agent (root in packages, uid 1002 in Docker)         nginx :443
 └─ otel-collector  (agent type OTEL_COLLECTOR,           └─ /otlp/  auth_request → rule "/otlp/": admin
    runs as user pmm-agent, see §6.1)                          │ proxy_pass http://127.0.0.1:4318/
     receivers: filelog/<source-id> per file source            ▼
                journald/<source-id> per journald src    otel-collector (supervisord program, user pmm)
                otlp (only after `add traces`, Phase 2)    receivers: otlp http 127.0.0.1:4318 (no gRPC)
     processors: memory_limiter → resource(identity)       processors: memory_limiter → transform(redaction)
                 → batch                                               → batch
     exporter: otlphttp → https://<server>/otlp/            exporter: clickhouse (user otel_writer, db otel)
       sending_queue on file_storage (disk)                     │
     extension: file_storage (paths.otelcol_data_dir)           ▼
                                                         ClickHouse  otel.logs (Phase 1)
PMM Server node (its own pmm-agent):                                 otel.otel_traces (+2 helpers, Phase 2)
 └─ otel-collector with the server log sources           ▲
    → http://127.0.0.1:4318 directly (no nginx hop)      │ pmm-managed (leader only): retention, size cap,
                                                         │ watermark: ALTER TABLE otel.* DROP PARTITION
                                                         │
                                         Grafana ── datasource "ClickHouse-OTEL" (user otel_reader)
                                         (fork middleware: Admins only)  ── Logs dashboard
```

**Identity contract** (ADR-02, PMM-15582) [V, ticket]. pmm-managed writes these into the `resource` processor of the node collector's config. Log-source config is never copied onto records.

| Attribute | Set on | Value |
|---|---|---|
| `pmm.node_id` | every record | the node of the pmm-agent that runs the collector |
| `pmm.agent_id` | every record | the OTEL collector agent's id |
| `pmm.service_id`, `pmm.service_name` | records from a service-bound source | the PMM service |
| `service.name` | app sources and server components; senders set it themselves for OTLP | the application or component name, never the PMM service |
| `log.file.path` | file sources | set by filelog `include_file_path: true` |

**Table layout** (ADR-04, see D4). Keep the upstream exporter's column set, so the exporter and the Grafana plugin work unchanged. Add materialized identity columns and sort by them:
- `PmmNodeId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.node_id']`
- `PmmServiceId LowCardinality(String) MATERIALIZED ResourceAttributes['pmm.service_id']`
- `ORDER BY (PmmNodeId, PmmServiceId, ServiceName, TimestampTime)`
- `PARTITION BY toDate(TimestampTime)`

---

## 4. The prototype: what to keep and what not to copy

**Keep** [V]:
- **Server shape:** a supervisord program behind nginx `/otlp/`, writing to the dedicated ClickHouse database `otel`, with `create_schema: false`.
- **Schema:** DDL that mirrors the upstream exporter (codecs, bloom and tokenbf indexes, `ttl_only_drop_parts`), and the cluster pattern copied from qan-api2 (`ReplicatedMergeTree`, `ON CLUSTER`).
- **Delivery:** the collector as a pmm-agent `AgentProcess`. pmm-managed renders the config into `TextFiles`, and pmm-agent fills in the server URL and credentials (the vmagent precedent, `managed/services/agents/vmagent.go:52-54`).
- **Presets:** an API with an "in use" guard, and the built-in operator chains as a starting set (mysql_error, nginx, grafana, pmm logfmt, postgres, clickhouse, supervisord). Re-test every one.
- **UI:** a Settings → OTEL tab using TanStack Query hooks.

**Don't copy** [V]:
1. **Receivers on `0.0.0.0:4317/4318`** on the server and on every client. Every monitored host becomes an unauthenticated relay that forwards with the agent's credentials.
2. **`"/otlp": viewer`** in the auth rules. The preset RPCs have no rule and only work through the `grafanaAdmin` fallback.
3. **Fixed `{{ }}` template delimiters** with preset YAML and paths pasted into the agent template. A preset containing `{{ .server_auth_b64 }}` would copy the agent's credentials into every log record.
4. **Config built by string concatenation** from user YAML, with no operator allow-list. On top of that, a heuristic "normalizer" rewrites the YAML users paste, and is duplicated in TypeScript.
5. **Log sources stored as JSON** in the agent's `custom_labels`. They become resource attributes on every record, and the UI's camelCase converter mangles the keys.
6. **The ClickHouse superuser shared** by the collector, Grafana and ADRE. Its password sits in a 0644 file.
7. **Retention set only at CREATE.** Changing it later does nothing, and DDL runs on every `UpdateConfiguration` with `context.Background()`.
8. **`start_at: end` with no `file_storage`**, so lines are lost across restarts. `memory_limiter` is not first in the pipeline, and multi-line entries are not handled.
9. **No version gate.** An older pmm-agent logs "unhandled agent type" and does nothing (`agent/agents/supervisor/supervisor.go:628-633,1164-1165` on `main`).
10. **The prebuilt `otelcol-contrib` 0.148.0** downloaded without a checksum, and two ~95 MB `pmm-managed` binaries committed.
11. **A textbox dashboard variable interpolated into SQL** (`'${trace_id}'`). There is also an unsigned service-map panel plugin.
12. **Migrations numbered 127–142,** which collide with `main` (latest is 120, `managed/models/database.go:1202`).

---

## 5. Findings on `main` that change the tickets

| # | Finding [V] | Where | Consequence |
|---|---|---|---|
| F1 | Packaged pmm-agent runs as **root**: the unit has no `User=`. Child processes inherit its uid, because `exec.Cmd` gets no `SysProcAttr.Credential`. | `build/packages/config/pmm-agent.service`; `agent/agents/process/process.go:150-178` | "Run as `pmm-agent`" (PMM-15572) needs a new privilege drop in pmm-agent, for this one process type, chosen on the client ([§6.1](#61-running-the-collector-as-pmm-agent)). |
| F2 | The pmm-agent temp dir is **wiped** on every start and on every re-render. | `agent/commands/run.go:101,156`; `agent/utils/templates/template.go:65-74` | Checkpoints and the disk queue need a persistent `paths.otelcol_data_dir`, like `paths.nomad_data_dir` (`agent/config/config.go:128,304-306`). |
| F3 | Child processes get an **empty environment**, and `Dir` is `/`. | `process.go:160-164` | The journald receiver must be given an absolute `journalctl` path, or a `PATH`. |
| F4 | An older pmm-agent **silently ignores** unknown agent types. | `supervisor.go:628-633,1164-1165` | The server must withhold the type and reject `add` (`models.PMMAgentSupported`, `managed/models/agent_helpers.go:1143-1151`; `managed/services/agents/state.go:304-309`). |
| F5 | The JSON gateway **drops unknown fields** (`DiscardUnknown: true`). So a new pmm-admin sending `collect_logs` to an old server gets no error. | `managed/cmd/pmm-managed/main.go:400-406` | pmm-admin checks the server version from local pmm-agent status before sending new fields (`admin/agentlocal/agentlocal.go:77`). |
| F6 | `pmm-admin remove` takes positional arguments `<service-type> <service-name>`. | `admin/commands/management/remove.go:41-45` | `remove logs` and `remove traces` need a restructured command (D9). |
| F7 | PMM Server's inventory fixtures run **only on first boot**. | `managed/models/database.go:1737-1748` | The server node's collector is ensured on connect (the Nomad pattern, `managed/services/agents/registry.go:353-405,526-545`), not in fixtures. |
| F8 | nginx strips the body from the auth subrequest. | `build/ansible/roles/nginx/files/conf.d/pmm.conf:133-136` | PMM-15576's plan to "reject queries to the OTEL datasource from non-Admins" in `auth_server` can't see which datasource a `/graph/api/ds/query` body targets. Enforce it in the Grafana fork instead, as a plugin-client middleware next to the existing `X-Proxy-Filter` forwarder (`grafana/pkg/services/pluginsintegration/pluginsintegration.go:222-223`). |
| F9 | Every node's service account has the **Admin** role. | `managed/services/grafana/client.go:657-674` | `"/otlp/": admin` admits pmm-agents (PMM-15571 holds). Any node can claim any `pmm.node_id`, which matters for LBAC later ([§7](#7-lbac)). |
| F10 | pmm-managed already has a privileged ClickHouse client. | `managed/cmd/pmm-managed/main.go:1098` | The leader-only retention, cap and watermark job can live in pmm-managed without a new ClickHouse user (D5). |
| F11 | QAN retention runs in qan-api2 on **every replica**. That is harmless only because a time-based `DROP PARTITION` is idempotent. | `qan-api2/main.go:290-303,447` | A size-cap or watermark drop is not idempotent across replicas. It must run on the leader only (PMM-15594), so not in qan-api2. |
| F12 | Server inventory (Agents tab) is rendered in the **Grafana fork**, not in `ui/`. | `grafana/public/app/percona/inventory/Inventory.types.ts:28-50` | A new agent type also needs a Grafana fork PR, or the Agents tab shows an unknown type. |
| F13 | The upstream exporter fills `ServiceName` from `service.name`. | exporter README [I] | PMM-15571's "`ServiceName` column filled from the identity attributes" would clash with it. Use `PmmServiceId`/`PmmNodeId` materialized columns, and keep `ServiceName` = `service.name` (D4). |
| F14 | PMM Server's own pmm-agent is set up with `--skip-registration` and has **no token**. It is admitted only for Connect and RTA from 127.0.0.1. | `build/docker/server/entrypoint.sh:302-311`; `auth_server.go:567-609` | The server node's collector can't authenticate to `/otlp/`. It must send to `http://127.0.0.1:4318` directly, which PMM-15575 already intends. |
| F15 | qan-api2 migrations are bound to the DSN database (`pmm`). There is one `schema_migrations` table, and `createDB` is the only `ON CLUSTER` DDL in the repo. | `qan-api2/migrations/migrations.go:86-142`; `qan-api2/db.go:114-148` | The `otel` database needs its own embedded migration chain (`qan-api2/migrations/otel/sql/`) with its own `x-migrations-table` (`otel_schema_migrations`), and a create-database step modelled on `createDB` (Replicated engine on clusters). The latest QAN migration is `22_…`. |
| F16 | pmm-managed never reloads nginx. Env-conditional locations are rendered by `entrypoint.sh` at start. | `pmm.conf:155`; `entrypoint.sh:195-262` | `/otlp/` is a **static** location. While the collector is stopped it answers 502, and node collectors keep the data in their disk queue and retry. |
| F17 | When a feature is turned off, the dynamic supervisord path deletes the `.ini` but doesn't run `supervisorctl update`. A running program keeps running. | `managed/services/supervisord/supervisord.go:125-131` | When `otel-collector` is turned off, pmm-managed must also run `update otel-collector` (or `StopSupervisedService`). |
| F18 | The shipped release notes say PMM does not accept inbound OpenTelemetry traffic. | `documentation/docs/release-notes/3.8.0.md:162`, `3.8.1.md:139` | The release notes and security docs for the release that ships `/otlp/` must say this has changed. |
| F19 | No disk, size-cap or watermark logic exists for ClickHouse anywhere. | grep `system.disks`, `bytes_on_disk` | The cap and the watermark are new code. Nothing exists to reuse. |
| F21 | PMM Server runs PostgreSQL 18 with `logging_collector=off`, and supervisord writes its output to `/srv/logs/postgresql.log`. | `managed/services/supervisord/pmm_config.go:106-128` | PMM-15575's `postgresql14.log` is stale. The server source is `postgresql.log`, and a test ties the source list to the shipped log files. |
| F20 | In HA, every pmm-server pod runs the same supervisord programs, and HAProxy sends external traffic to the leader only. Leader-only work is registered with `haService.AddLeaderService`. | `percona-helm-charts/charts/pmm-ha/templates/haproxy-configmap.yaml:55-61`; `managed/cmd/pmm-managed/main.go:1247-1324` | Every pod runs `otel-collector`; only the leader's gets client traffic, and each pod's own pmm-agent ships its own server logs. The cleanup job is a leader service. |

---

## 6. Design details that the tickets leave open

### 6.1 Running the collector as `pmm-agent`

pmm-agent decides this locally; the server never chooses a uid:
- If pmm-agent's effective uid is 0 and a user named `pmm-agent` exists, start the `OTEL_COLLECTOR` process with `SysProcAttr.Credential{Uid, Gid, Groups}`. The groups are `pmm-agent`'s supplementary groups as read from the OS, so packaging adds `adm` and `systemd-journal` (PMM-15577).
- If pmm-agent is not root (Docker image, uid 1002), run the collector as pmm-agent's own user.
- If pmm-agent is root and the user doesn't exist, refuse to start the collector, and report the reason in its status.

Other agent types keep their current behaviour. The collector reaches the network only through `otlphttp`. The OCB build includes no `file` exporter and no other component that writes files, so a compromised config can't write files outside `file_storage`.

### 6.2 Log sources

- **Storage:** their own PostgreSQL table, `log_sources`:
  - `id`, `otel_collector_agent_id` (FK, `ON DELETE CASCADE`)
  - `kind` (`file`/`journald`), `path`, `units`, `preset_id` (FK, `ON DELETE RESTRICT`)
  - `service_id` (FK, `ON DELETE CASCADE`), `app_name`
  - `discovery` (`""`, `mysql_error`, `mysql_slow`, `mysql_general`, `postgresql`, `mongodb`)
  - `custom_labels`, `status`, `status_reason`
  - `created_at`, `updated_at`
  - unique (`otel_collector_agent_id`, `kind`, `path`, `units`)
  - check: `service_id` and `app_name` are never both set
- **One collector per pmm-agent:** a partial unique index on `agents(pmm_agent_id) WHERE agent_type = 'otel-collector'`, not a check-then-insert as in the prototype.
- **Status flow:** pmm-agent reports `status` and `status_reason` for each source in a new repeated field on `StateChangedRequest`. For example: a path outside the allow-list, a file it can't read, or `journalctl` missing.
- **Allow-list:** pmm-agent checks paths before rendering. It drops sources that fail, and keeps the collector running for the rest.

### 6.3 Default log collection on `pmm-admin add`

- **Discovery runs on the client.** pmm-agent connects to the database and can `stat` the file.
- **Transport:** a new server→agent request, `LogDiscoveryRequest{service_type, dsn, kinds}` → `LogDiscoveryResponse{entries[]{kind, path, units, readable, reason}}`.
  - It is version-gated.
  - It is called inside the add transaction, after `ServiceInfoBroker`, with a 5 s timeout.
  - It is called again on every pmm-agent reconnect for sources whose `discovery` is set, so a moved file is followed (PMM-15584).
  - Reuse `slowlog.go:226-282` (relative path resolution against `@@datadir`) and `mongolog.go:241-267` (`getCmdLineOpts`).
- **Default:** the CLI default is `--collect-logs=error` for MySQL. For PostgreSQL and MongoDB it is `--collect-logs` on (one log each). A remote service gets no source, and the command prints why. `none` opts out.
- **Why not guess default paths:**
  - `@@log_error`, PostgreSQL settings and `getCmdLineOpts` are readable with the privileges PMM already documents.
  - A guessed path is wrong whenever two instances share a host.
  - A guessed path is also wrong under custom layouts, and it attributes the wrong file to a service without saying so.
- **Old server:** pmm-admin doesn't send `collect_logs` to a server older than the release that adds it, and prints that logs need a newer server (F5).

### 6.4 Parsers in the UI

- Presets are added in Settings → OTEL → Parser presets. Built-ins are read-only and can be cloned.
- Settings → OTEL → Log sources lists every node's sources with their status. The same tab adds, edits and removes a source (node, file or journald, path, preset, owner) through the same `LogSourcesService` API that pmm-admin uses. The path is still checked against the node's allow-list, so the UI can't widen what a node exposes.
- A change reaches the node's collector through the normal `SetState` push. Only the agents that use the preset get it.
- **Not in scope:** a preview that runs a preset on sample lines. It needs a parser on the server, either the stanza library in pmm-managed or a debug pipeline in the collector. Do it as a follow-up if users ask for it.

### 6.5 Retention, size cap and watermark

- Retention is enforced by a **daily partition drop**, as QAN does, run by pmm-managed on the leader. The same job enforces `max_disk_gb` and the watermark by dropping the oldest `otel` partitions first. One mechanism covers all three limits, and a retention change applies to stored data on the next pass, with no `MODIFY TTL` and no part rewrites (D1).
- **A single day bigger than the cap.** If only today's partition is left and the cap is still exceeded, pmm-managed stops `otel-collector` and raises a health warning. The node queues buffer on disk until the next pass, or until an admin raises the cap (D6).

### 6.6 Who renders the node collector's config

PMM-15572 says pmm-managed "generates the collector config and pushes it". Taken literally, the agent receives opaque YAML. Then the agent can't apply its allow-list per source without parsing that YAML, and every preset and path passes through the agent's `text/template`, which is the prototype's credential-leak path (§4, item 3).

**Decision (D11):**
- pmm-managed decides everything and sends it as typed data: a new `OtelCollectorParams` message in `SetStateRequest.AgentProcess` (`api/agent/v1/agent.proto:56-66`). It carries the sources, the preset operator chains (validated on the server), the identity attributes, the queue size and, from Phase 2, the traces receiver.
- pmm-agent turns that data into collector YAML with typed Go structs and `yaml.v3`. No `text/template` is involved.
  - It checks each source against its allow-list, and readability, and leaves out the sources that fail.
  - It fills in the exporter endpoint and credentials from its own config.
  - It escapes every `$` in user-supplied strings as `$$`, so the collector's `${env:…}` and `${file:…}` expansion can't run.
  - It reports a status for each source.
- This keeps the contract of PMM-15572 (the server owns content; nothing is configured by hand on the node) and removes the template-injection class entirely.

### 6.7 Built-in presets

- Built-in presets live as YAML files in `managed/services/otel/presets/builtin/`, embedded with `go:embed` and versioned with the code.
- At startup pmm-managed upserts them into `log_parser_presets` with `built_in = true`, so log sources reference every preset by foreign key, and `usage_count` is a plain `COUNT`.
- If a new release adds a built-in whose name an existing custom preset already uses, pmm-managed renames the custom one to `<name>_custom` and logs a warning. Sources keep their foreign key, so nothing changes for them (D13).

### 6.8 Settings in HA

Nothing on `main` propagates a settings change to other replicas. Each replica applies config only for its own requests (`managed/services/server/server.go`). PMM-15594 needs OTEL settings to apply on every replica.

**Decision (D14):** each replica runs a small reconcile loop (`otel.Service.Run`). Every 30 s it reads the settings row and calls `UpdateConfiguration` when the OTEL part has changed. The cleanup job (§6.5) is a leader service and reads the settings on every pass anyway.

---

## 7. LBAC

### 7.1 How it works today [V]

- **Roles:** `roles(id, title, filter, description)` and `user_roles` (migrations 73 and 76, `managed/models/database.go:822-869`).
  - A filter is a PromQL series selector; matchers inside one role are ANDed, and a user's roles are ORed.
  - A role with an empty filter means full access.
  - LBAC is switched on by `settings.access_control.enabled` or `PMM_ENABLE_ACCESS_CONTROL`. It is on by default in the pmm-ha chart.
- **Header:** for paths in `lbacPrefixes` (`managed/services/grafana/auth_server.go:146-160`), the nginx `auth_request` handler returns the user's selectors in the `X-Proxy-Filter` header.
- **Enforcement:**
  - vmproxy replaces `extra_filters[]` for VictoriaMetrics.
  - qan-api2 turns the selectors into a ClickHouse `WHERE` clause.
  - QAN can do this because pmm-managed denormalizes the service and node labels into every QAN row at ingest (`managed/services/qan/client.go:288-411`).
- **Labels that can be filtered** come from `models.MergeLabels(node, service, agent)` (`managed/models/models.go:58-86`):
  - node: `node_id`, `node_name`, `region`, `az`, …
  - service: `service_id`, `service_name`, `environment`, `cluster`, `replication_set`, …
  - plus custom labels.

### 7.2 What this means for logs and traces

- An OTel record carries only IDs (`pmm.node_id`, `pmm.service_id`). The labels LBAC filters on live in PostgreSQL.
- Grafana OSS lets every org member query every datasource [I].
- A ClickHouse datasource takes raw SQL, so a proxy can't safely append `WHERE` clauses to it.
- nginx can't inspect the query body (F8).
- **Conclusion: LBAC can't be enforced on a Grafana ClickHouse datasource.** It needs a PMM API that builds the SQL itself.

### 7.3 Phase 1 (this epic): Admin-only, enforced in three places

1. **Ingest:** `"/otlp/": admin` in the auth rules. Nobody else can write.
2. **Read:**
   - The Grafana fork denies `QueryData` and `CallResource` for the datasource UID `pmm-otel` unless the user's org role is Admin.
   - The Logs dashboard sits in a folder with Admin-only permissions.
   - The `otel_reader` ClickHouse user has `SELECT` on `otel.*` only.
3. **Content:** a redaction `transform` on the server masks bearer and `glsa_` tokens, `Authorization` headers and `password=` values before they are stored.

A user restricted by LBAC sees no logs in Phase 1. That is fail-closed, and it matches the epic.

### 7.4 Phase 3 (follow-up epic): LBAC for logs and traces

**Recommended: a logs query API with an ID allow-list, plus identity checked at ingest.**

1. **Ingest identity.**
   - The auth server returns the authenticated node's id for a per-node service-account token. nginx passes it on as `X-PMM-Node-Id`.
   - The server collector's OTLP receiver keeps request metadata (`include_metadata: true`). A `resource` processor overwrites `pmm.node_id` from it [I; verify in the spike].
   - Shared non-token credentials can't be checked. Those records are marked `pmm.identity_verified=false`.
2. **Query API.**
   - `LogsService.Search` and `TracesService.Search` in qan-api2 or pmm-managed, under `/v1/logs` and `/v1/traces`. Both are in `lbacPrefixes` with an nginx location that sets `X-Proxy-Filter`.
   - Each request takes structured filters only (time range, node, service, app, severity, text) and never raw SQL.
3. **Resolution.**
   - Parse the selectors with the Prometheus parser.
   - Evaluate them with `labels.Matcher.Matches` against `MergeLabels(node, svc, nil)` for each service, and `MergeLabels(node, nil, nil)` for each node. A missing label counts as `""`, the VictoriaMetrics semantics, including anchored regexes.
   - Bind the resulting ID lists as query parameters: `PmmServiceId IN {s} OR (PmmServiceId = '' AND PmmNodeId IN {n})`.
4. **Fail closed.**
   - No header for a non-Admin means deny.
   - Any selector that doesn't parse means deny.
   - Service-account tokens get an explicit decision rather than today's skip.
5. **UI.** A log explorer and a trace view in the PMM UI built on this API. The Grafana `pmm-otel` datasource stays Admin-only.
6. **Semantics to document.**
   - Access follows current labels. Revoking a role hides old logs at once, and the logs of a deleted service become invisible to restricted users.
   - Node-level logs are visible only through node labels, as node metrics are today.
   - A trace that crosses services shows only the spans the user can see.

**Rejected options:**
- **ClickHouse row policies per PMM role.** They need per-request identity, ClickHouse access management, and sync on every inventory change. Keep them only as possible defence in depth under the API.
- **A SQL-rewriting proxy in front of ClickHouse.** Any divergence between its parser and ClickHouse's is a bypass.
- **Grafana-side controls only.** They are not a security boundary.

Plan 06 holds the task list. It is a design-level plan, to be re-planned once Phase 2 lands.

---

## 8. Decisions

| # | Decision | Recommendation | Blocks |
|---|---|---|---|
| D1 | Retention mechanism (ADR-05) | A daily partition drop by pmm-managed on the leader. The same job enforces the cap and the watermark. | Plan 01 |
| D2 | `--collect-logs` default | On for local services (MySQL: `error`; PostgreSQL and MongoDB: their one log), with `none` to opt out. Slow and general logs stay opt-in. This changes the epic, so the epic owner must agree. | Plan 04 |
| D3 | Collector uid when pmm-agent is root | pmm-agent drops to user `pmm-agent` for this process type only ([§6.1](#61-running-the-collector-as-pmm-agent)). | Plan 02 |
| D4 | Columns and sort key (ADR-04; it can't change after release) | Keep `ServiceName` = `service.name`. Add `PmmNodeId` and `PmmServiceId` as materialized columns. `ORDER BY (PmmNodeId, PmmServiceId, ServiceName, TimestampTime)`. Database sources set `service.name` to the program (`mysqld`, `postgres`, `mongod`). | Plan 01 |
| D5 | Where the cleanup job runs | pmm-managed, leader only, using the existing ClickHouse client (F10, F11). | Plan 01 |
| D6 | A single day larger than the cap | Stop `otel-collector` and raise a health warning; clients buffer. | Plan 01 |
| D7 | UI scope for log sources | Add, edit and remove sources in Settings → OTEL through the same API; checked against the allow-list. | Plan 03 |
| D8 | Profiles and eBPF | No profiles work until the ClickHouse exporter supports them; record this as an ADR. eBPF stays in PMM-15588. | Roadmap |
| D9 | `pmm-admin remove logs` shape | Make `remove` a Kong command group: `remove logs`, `remove traces`, and a default service form that keeps today's positional syntax. If Kong can't combine a default positional form with subcommands, add `logs` and `traces` to the service-type enum instead. Check with a spike in plan 04, task 1. | Plan 04 |
| D10 | Where the server's own logs are collected | Through the server node's own pmm-agent collector, sending to `127.0.0.1:4318` (PMM-15575). This avoids a second filelog config in supervisord. | Plan 02 |
| D11 | Who renders the node collector's YAML | pmm-agent renders it from typed `OtelCollectorParams` sent by pmm-managed, with no templates ([§6.6](#66-who-renders-the-node-collectors-config)). This departs from the literal wording of PMM-15572. | Plan 02 |
| D12 | API home | One new domain, `api/otel/v1` (`OtelService`), under `/v1/otel/…`: presets, log sources, discovery, status and purge. Settings stay in `/v1/server/settings`. This differs from PMM-15574 (`/v1/server/log-parser-presets`) and PMM-15571 (`/v1/server/otel:purge`), and gives one `"/v1/otel": admin` rule. | Plans 01, 02 |
| D13 | Built-in and custom presets | One table. Built-ins are upserted from embedded files at startup, and a clashing custom preset is renamed ([§6.7](#67-built-in-presets)). | Plan 02 |
| D14 | Settings in HA | A per-replica reconcile loop for the OTEL settings ([§6.8](#68-settings-in-ha)). | Plan 01 |
| D15 | Node collectors when OTEL is turned off | Every collector stops, the server's and every node's (`IgnoreOtelCollector`). File checkpoints are kept, so turning it back on resumes each file where it stopped. | Plan 02 |
| D16 | When the server side of traces lands | Phase 2, together with the node receiver, as the epic says. | Plan 05 |
| D17 | Existing access gaps found during research | Fixed in this programme, in plan 07, worded as fixes without exploit detail. | Plan 07 |

**Interview of 2026-10-10:**
- Every recommendation above was accepted: D1–D14, and D15–D17 as written.
- D8 is decided as "defer profiles and record ADR-17".
- The ADRs follow PMM-15590's numbering:
  - ADR-01..14 as in the ticket; ADR-15 and ADR-16 stay reserved for eBPF (PMM-15588).
  - Then ADR-17 (profiles deferred), ADR-18 (LBAC for OTel data), ADR-19 (default database log collection, D2) and ADR-20 (node collector config rendered by pmm-agent, D11).
- ADR-19, ADR-20, and the D12 path change differ from the epic as written. They stay **Proposed** until the epic owner and the architects accept them.

---

## 9. Risks

- **Log volume shares ClickHouse and its disk with QAN on the 8 GB profile.** The cap, the watermark and the separate ClickHouse users with resource limits are release blockers (PMM-15571, PMM-15592).
- **On by default after upgrade** means the first upgrade starts a new process, creates a schema and turns on collection of the server's own logs everywhere at once. The capacity test (PMM-15592) must run before the default is final.
- **Discovery doesn't work in Kubernetes sidecars:** the uid is 1002, there is no `journalctl`, and database log directories are not mounted. The operators must mount them (PMM-15577). Until they do, document that `--collect-logs` reports "file not visible" there.
- **Non-root reads:** on RHEL, MySQL, PostgreSQL and MongoDB logs are not readable by `adm` [I, distro defaults]. Users must add an ACL or group. The `status_reason` and the docs must say how.
- **A schema written by qan-api2 and a collector built from a pinned version can drift.** A CI contract test runs the pinned collector against the migrated schema (PMM-15571).
- **The work is large.** 21 tickets across pmm, grafana, the helm charts and the build. The prototype took 265 commits without review. Follow the roadmap's order, and merge ADRs and the threat model first.

## 10. Security-sensitive findings

The research turned up four access-control gaps in existing surfaces on `main`, outside this epic. At the requester's direction they are fixed in [plan 07](../plans/2026-10-10-opentelemetry-07-access-hardening.md). Following `AGENTS.md` (keep exploit detail out of public places), plan 07 says what to change and how to test it, not how to abuse the gaps. Per `SECURITY.md`, they should also be filed in the PMM Jira project, so that Percona's fix timelines apply.

---
name: mongodb-incidents
description: Percona_MS_MongodbInstanceNotAvailable, Percona_MS_MongodbPrimaryRoleHighFlipping,
  Percona_MS_MongodbReplSetHasNoPrimary, Percona_MS_MongodbConnections, Percona_MS_MongodbReplStateUnknown,
  Percona_MS_MongodbChunksImbalance, Percona_MS_MongoDBReadWriteQueueHigh, Percona_MS_MongoDBOpcounters,
  Percona_MS_MongodbReadWriteWTTicketUse, Percona_MS_MongodbReplLag, Percona_MS_MongoDBOplogWindow,
  Percona_MS_MongodbWTCheckpointTimeHigh, Percona_MS_MongodbWTDirtyRatio, Percona_MS_MongodHighCursorCount,
  Percona_MS_MongoDBHighCacheMissRatio, Percona_MS_MongoDBHighHeapUsage, Percona_MS_MongoDBHighFlowControl,
  Percona_MS_MongoDBHighWriteConflict, Percona_MS_MongoDBTLSExpiry, Percona_MS_InconsistentIndexes
  — MongoDB replica set, sharding, WiredTiger, oplog, TLS
---

# MongoDB incidents (managed service alerts)

## Goal

Route the MongoDB alert to the correct checks using PMM (inventory, Grafana, Prometheus, QAN/logs), then list exact **mongosh** / operator commands for replica set, sharding, WiredTiger, and TLS issues.

## Workflow

### Alert routing

| Alert | Focus |
|--------|--------|
| **Percona_MS_MongodbInstanceNotAvailable** | `mongod` / process down; connectivity |
| **Percona_MS_MongodbPrimaryRoleHighFlipping** | Frequent elections — network/partials |
| **Percona_MS_MongodbReplSetHasNoPrimary** | No writable primary — quorum / split |
| **Percona_MS_MongodbConnections** | Connection storms vs limits |
| **Percona_MS_MongodbReplStateUnknown** | Member state not readable |
| **Percona_MS_MongodbChunksImbalance** | Sharded cluster balancer / chunk distribution |
| **Percona_MS_MongoDBReadWriteQueueHigh** | Operation backpressure |
| **Percona_MS_MongoDBOpcounters** | Sudden op rate change |
| **Percona_MS_MongodbReadWriteWTTicketUse** | WiredTiger ticket exhaustion |
| **Percona_MS_MongodbReplLag** | Secondary replication lag |
| **Percona_MS_MongoDBOplogWindow** | Oplog too small for recovery/catch-up |
| **Percona_MS_MongodbWTCheckpointTimeHigh** | Checkpoint duration — IO or cache pressure |
| **Percona_MS_MongodbWTDirtyRatio** | High dirty cache — flush / IO |
| **Percona_MS_MongodHighCursorCount** | Open cursors leak |
| **Percona_MS_MongoDBHighCacheMissRatio** | Working set > RAM |
| **Percona_MS_MongoDBHighHeapUsage** | mongod heap pressure |
| **Percona_MS_MongoDBHighFlowControl** | Replication flow control active |
| **Percona_MS_MongoDBHighWriteConflict** | WiredTiger write conflicts |
| **Percona_MS_MongoDBTLSExpiry** | Certificate nearing expiry |
| **Percona_MS_InconsistentIndexes** | Index build / consistency issue |

1. **Scope**
   * **pmm-inventory** → **service_id**, **node_id**, **agent_id** for the MongoDB service from alert labels (replica set name, host).

2. **PMM metrics and panels**
   * **pmm_observability_map**: `engine=mongodb`, `service_id` from step 1. Pick **intent** by alert:
     | Alert cluster | intent |
     |---|---|
     | InstanceNotAvailable | availability |
     | Connections | connections |
     | ReplLag, FlowControl, OplogWindow, Primary flipping, NoPrimary, ReplStateUnknown | replication |
     | Opcounters, ReadWriteQueue | workload |
     | WTTicketUse, WTCheckpoint, WTDirty, CacheMiss, Heap, WriteConflict | memory or latency |
   * Render **2–4** panels from map; **embed every `image_url`** in the answer. **`pmm_metrics_snapshot`** on panel `expr`.
   * **`pmm.metrics`** top patterns by `service_id` when workload/connections alerts correlate with query load. For per-operation plan/index analysis (explain, COLLSCAN, `$indexStats`) use the **mongodb-query-analysis** skill.

3. **Logs**
   * **otel.logs** for `mongod`, election, rollback, TLS, index build errors in the alert window.

4. **Branch: availability / no primary / flipping**
   * Operator **mongosh**:
   ```javascript
   rs.status()
   rs.printSecondaryReplicationInfo()
   ```
   * Check member **stateStr**, **health**, **lastHeartbeat**, **optimeDate** drift.
   * Network partitions, clock skew (see **pmm-monitoring-health**), or resource saturation (**node-capacity-saturation**) often correlate.

5. **Branch: replication lag / flow control / oplog**
   * Lag and secondary progress:
   ```javascript
   rs.printSecondaryReplicationInfo()
   db.printReplicationInfo()
   ```
   * Oplog window too small: increase oplog size (planned maintenance) or reduce write burst — exact steps depend on deployment (mongod.conf / Kubernetes CR); state requirement for rolling change.
   * Flow control: identify primary saturation; scale or reduce write load; check `replSetGetStatus` flow control fields in server version docs.

6. **Branch: connections**
   * In mongosh:
   ```javascript
   db.serverStatus().connections
   db.currentOp({ "active": true })
   ```
   * Correlate with app pool sizing; **pmm.metrics** QAN for Mongo if configured for query patterns.

7. **Branch: WiredTiger (tickets, checkpoints, dirty, cache miss, write conflicts)**
   * `serverStatus` wiredTiger section:
   ```javascript
   db.serverStatus().wiredTiger
   ```
   * If cache pressure: RAM sizing, `storage.wiredTiger.engineConfig.cacheSizeGB`, workload (large scans), index design.
   * Write conflicts: shorter transactions, idempotent retries, schema/index tuning — pair with QAN and logs.

8. **Branch: sharding / chunk imbalance**
   * On mongos (operator):
   ```javascript
   sh.status()
   ```
   * Balancer enabled, jumbo chunks, failed migrations — follow internal sharding runbook; **Balancer** window and zone tags.

9. **Branch: high cursors**
   ```javascript
   db.serverStatus().metrics.cursor
   db.currentOp({ "waitingForLock": true })
   ```
   * Find long-running `getMore` / unclosed cursors from app side.

10. **Branch: TLS expiry** (`MongoDBTLSExpiry`)
    * List cert paths from deployment; operator checks:
    ```bash
    openssl x509 -in /path/to/server.pem -noout -dates
    ```
    * Renew cert before expiry; restart `mongod` per rolling procedure.

11. **Branch: inconsistent indexes**
    * Check logs for index build failures; `db.currentOp()` for index builds; may require rebuild per MongoDB docs — coordinate maintenance window.

## Synthesize Findings

State whether the issue is topology (elections, primary), capacity (CPU/IO/RAM), replication (lag, oplog, flow control), sharding (balancer), WiredTiger, client behavior (cursors, pools), or TLS/config. Link evidence from panels and logs.

### Environment availability matrix (shared)

Before prescribing a command, decide **which surfaces this deployment exposes**. PMM tools (inventory, observability map, metrics snapshot, QAN on `pmm.metrics`, `otel.logs`, Grafana render) are available in **every** environment — always try those first. Host shell, DB superuser, and config-file edits are **not** universal.

| Surface | Self-managed (VM / bare-metal / Docker) | Kubernetes / Operator (PSMDB, PXC, PG operator) | Managed cloud (RDS / Aurora / Cloud SQL / Atlas) |
|---|---|---|---|
| PMM metrics / QAN / observability map | ✅ | ✅ | ✅ (exporter may be **remote/cloud-mode**) |
| `otel.logs` (PMM-collected) | ✅ | ✅ | ⚠️ partial — DB logs often only in the cloud console |
| Host shell (`journalctl`, `grep /var/log/...`, `openssl`, `df`) | ✅ operator | ⚠️ via `kubectl exec` into the pod, not the node | ❌ none |
| DB shell (`psql`, `mongosh`, `mysql`) | ✅ operator | ✅ operator via `kubectl exec` | ✅ but **no superuser** (rds_superuser / atlasAdmin only) |
| `ALTER SYSTEM` / `mongod.conf` / `my.cnf` edits | ✅ | ❌ — change the **Custom Resource**, not the file (operator reverts file edits) | ❌ — change a **parameter group / cluster config**, then reboot/apply |
| `pg_terminate_backend`, `KILL`, force-primary, oplog resize | ✅ with approval | ✅ with approval | ⚠️ often restricted or wrapped by a cloud API |

**Routing rules:**

- **Always lead with PMM/QAN/`otel.logs`.** They work everywhere and need no host/DB access.
- **Gate host-shell and superuser steps.** Label them **"operator, self-managed / k8s only"**. On managed cloud, **skip** them and use the cloud-native equivalent (console logs, Performance Insights / Cloud Monitoring, parameter groups, provider CLI).
- **Config changes:** on **operator** deployments edit the **CR** (e.g. `PerconaServerMongoDB`, `PerconaXtraDBCluster`, `PerconaPGCluster`) and let the operator roll it out; on **managed cloud** edit the **parameter group / flag** and apply per the provider; only on **self-managed** edit the file directly and reload/restart.
- **Detect the environment** from inventory (`service_type`, `node_model`/cloud labels, address like `*.rds.amazonaws.com`, `*.mongodb.net`) before recommending a step. If unknown, state the assumption and give the **PMM-only** path plus the operator path as an option.
- **Never tell the user to run a command their environment can't execute** (e.g. `journalctl` on RDS). Offer the reachable alternative instead.
### PMM observability and panel embed rules (shared)

**Evidence hierarchy:** inventory → **`pmm_observability_map`** → **`pmm_metrics_snapshot`** on panel `expr` → **`pmm_render_grafana_panel`** (embed PNGs, best-effort) → QAN + MySQL EXPLAIN → scoped fallbacks (`pmm_discover_series_labels`, `pmm_list_metric_names` prefix max 50) → **`execute_prometheus_range_query`** at most one per turn, last resort.

**Render failures are non-blocking:** 502, timeout, blank PNG, or curl error on render must **never** skip snapshot, QAN, EXPLAIN, or final synthesis. State **`rendered M/N`**; deliver analysis from snapshot + QAN even when **`rendered 0/N`**.

**FORBIDDEN:** unfiltered `GET /api/v1/label/__name__/values`, full dashboard JSON to the LLM, guessing panel IDs or metric names.

**Observability map:** `GET /v1/grafana/observability-map` via **`pmm_observability_map`**. Pass **`engine`** (`mysql`, `postgresql`, `mongodb`, `valkey`, `node`), **`intent`**, **`service_id`**. Use returned **`primary.dashboard_uid`**, **`panels[].id`**, **`panels[].expr`**. Fall back to **`pmm_list_dashboard_panels`** only when the map warns or intent is unknown (e.g. ProxySQL — no map route yet).

**Metrics snapshot:** **`pmm_metrics_snapshot`** on panel `expr` — returns server-computed **stats** (min/max/mean/median/p25/p75/p95/p99), **change_points**, **anomalies**; not raw matrices. Run **immediately after map** — does not require render. `start`/`end`: RFC3339 or Unix (not `now-24h`). Requires ADRE enabled.

**Render panels (best-effort when this skill uses Grafana):**
1. Inventory first: **`service_id`**, **`node_id`**, **`agent_id`**, **`service_name`**, **`node_name`**, **`version`** → pass all as `var_*` overrides to **`pmm_render_grafana_panel`**. Missing **`agent_id`** often causes blank/timeout renders.
2. Tool: **`POST /v1/grafana/render/resolve`** only (via **`pmm_render_grafana_panel`**).
3. Render **sequentially**; `from`/`to` without quotes (e.g. `now-6h`, `now`). Retry once with shorter window or fixed vars on failure — then **continue snapshot/QAN/EXPLAIN**.
4. **Embed every successful render** in the user-visible answer:
   ```markdown
   ![Panel title](/v1/grafana/render/blob/{hash}.png)
   [Open in Grafana](dashboard_url)
   ```
   Use **`image_url`** and **`dashboard_url` exactly** from tool JSON. Do not skip images for successful renders; do not use `dashboard_url` as img src, do not rebuild URLs.
5. State **`rendered M/N`** if any panel failed; **do not abort** the investigation.

**Scoped metric fallback (when map warns):** **`pmm_discover_series_labels`** with `service_id` + **`metric_prefix`** from map fallback, or **`pmm_list_metric_names`** with prefix only (max 50).

**QAN (database services):** For workload-correlated incidents, query **`pmm.metrics`** top slow patterns by **`service_id`** — see **`general`** skill for query shape and MySQL EXPLAIN Step 3b when applicable. **Always run** even when render fails.
## Recommended Remediation Steps

* **No primary / split**: restore network quorum; fix misconfigured votes; restart stuck members only per procedure; never force primary without runbook approval.
* **Lag**: reduce primary load; add secondaries; disk/IO upgrade; end long secondary batch jobs.
* **Oplog**: increase oplog (version-specific procedure); e.g. for some installs, rolling change with `replication.oplogSizeMB` and resync if required — document exact steps for the customer’s install type.
* **WT tickets / cache**: tune cache size; fix hot collections; add indexes to reduce full collection scans.
* **Connections**: raise limits only with sizing; fix connection leaks; deploy pooler pattern at app.
* **TLS**: renew certificates; update trust chain; rolling restart members.
* **Chunks**: enable balancer; split jumbo; fix tag zones; clear migration errors from logs.

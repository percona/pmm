# PMM‑15122 / PMM‑15195 — Percona Server / MySQL 9.7 support investigation & dashboard fixes

**Author:** automated investigation run
**Date:** 2026‑07‑08
**PMM Server:** 3.8.0 · **pmm‑agent (client):** 3.9.0‑v3 · **mysqld_exporter (embedded):** 0.17.2

---

## 0. Environment used for verification

| Target | Image | Version reported by `SELECT VERSION()` | Role |
|---|---|---|---|
| `ps80‑PS8.0` | `percona/percona-server:8.0` | **8.0.46‑37** (Percona Server) | *baseline — must not regress* |
| `ms97‑MySQL9.7` | `mysql:9.7` | **9.7.1** (MySQL Community) | *9.x target* |

> **Reality check — Percona Server 9.7 does not exist yet.** As of this run the newest
> Percona Server for MySQL is **8.4.10 / 8.0.46**; there is no `percona/percona-server:9.x`
> tag on Docker Hub. Upstream **MySQL 9.7** *does* exist (Innovation release), so it was used
> as the 9.x proxy. This matters for interpreting "missing" metrics (see §4): Percona‑specific
> collectors are absent on upstream MySQL by definition, **not** because of a PMM bug.

Both instances were registered with `pmm-admin add mysql --query-source=perfschema`. Both use
`caching_sha2_password` (the 9.x default; `mysql_native_password` is removed in 9.x) and the
embedded exporter connected cleanly to both — **auth is not a blocker**.

All 6 `mysqld_exporter` scrape targets (hr/mr/lr × 2 services) report `up=1`; the pmm‑agent log
contains **no mysqld_exporter errors and no "unsupported"/parse warnings** against 9.7 (only an
unrelated `node_exporter` udev warning).

---

## 1. PMM‑15122 Step 1 — embedded exporter

* pmm‑agent embeds **`mysqld_exporter` 0.17.2** (Percona build of `prometheus/mysqld_exporter`).
* Against MySQL 9.7 the exporter **runs and scrapes without error**: 863 distinct `mysql_*`
  metric families are produced (vs 967 on Percona Server 8.0).
* **Verdict:** the current exporter is *functionally compatible* with MySQL 9.7 scraping — no
  upgrade is strictly required to collect metrics. The gaps are in *dashboards* and in
  *Percona‑specific collectors* that depend on server features, not in the exporter core.

## 2. PMM‑15122 Step 3/4 — metric coverage delta (8.0 → 9.7)

| | count |
|---|---|
| distinct `mysql_*` metrics on PS 8.0 | 967 |
| distinct `mysql_*` metrics on MySQL 9.7 | 863 |
| present on 8.0, **missing on 9.7** | 135 |
| **new** on 9.7 | 31 |

The 135 "missing" split into two very different buckets:

### 2a. Genuine upstream removals (affect real Percona Server 9.7 too — **must fix in PMM**)
| Metric (removed 8.0.30 → 9.x) | Replacement present on 9.7 |
|---|---|
| `mysql_global_variables_innodb_log_file_size` | `mysql_global_variables_innodb_redo_log_capacity` |
| `mysql_global_variables_innodb_log_files_in_group` | `mysql_global_variables_innodb_redo_log_capacity` |
| `mysql_global_variables_innodb_numa_interleave` | *(none — config var removed)* |
| `mysql_global_variables_query_cache_size`, `have_query_cache=0` | *(feature removed in 8.0 — see §5)* |
| `mysql_global_status_qcache_*` | *(feature removed in 8.0)* |

### 2b. Percona‑Server‑only metrics (absent only because the proxy is *upstream* MySQL — **not a PMM bug**)
`Innodb_checkpoint_age`, `Innodb_checkpoint_max_age`, `Innodb_lsn_current`,
`Innodb_buffer_pool_pages_made_young`, and the whole
`information_schema.USER_STATISTICS / CLIENT_STATISTICS / TABLE_STATISTICS / INDEX_STATISTICS`
(`userstat`) family (~120 of the 135). On upstream MySQL 9.7,
`SELECT … FROM information_schema.TABLE_STATISTICS` → `ERROR 1109 Unknown table`. On a real
Percona Server 9.7 with `userstat=ON` these would return, so the panels that depend on them
would work. **Where an *upstream‑available* equivalent exists, PMM was made version‑robust
anyway (see §3)** so the dashboards degrade gracefully on non‑Percona 9.x.

## 3. PMM‑15195 — dashboard breakages, root cause, and the fix applied

Method: a harness ran **every panel query in each affected dashboard** against VictoriaMetrics
for both services and flagged panels that return data on 8.0 but nothing on 9.7. Panels that are
empty on *both* versions are idle‑DB/feature artifacts, not 9.7 regressions, and are listed as
backlog, not fixed here.

### ✅ Fixed in this change (InnoDB redo‑log / LSN / checkpoint family)
Root cause: these panels read **Percona status vars** (`Innodb_checkpoint_age/max_age`,
`Innodb_lsn_current`) or **removed 8.0.30 config vars** (`innodb_log_file_size *
innodb_log_files_in_group`). All are absent on 9.7. Upstream MySQL 8.0.30+/9.x exposes
equivalents that are present on **both** 8.0 and 9.7:
`mysql_global_status_innodb_redo_log_current_lsn`, `…_checkpoint_lsn`, and
`mysql_global_variables_innodb_redo_log_capacity`.

The fix appends each upstream equivalent as an **additive `or` fallback**. On Percona 8.0 the
original branch returns data and wins (**byte‑identical result → zero regression**); on 9.x the
original is empty and the fallback resolves.

| Dashboard | Panel | Fallback added |
|---|---|---|
| MySQL InnoDB Details | Total Redo Log Space | `… or innodb_redo_log_capacity` |
| MySQL InnoDB Details | Max Log Space Used | `… or (cur_lsn − ckpt_lsn)/redo_log_capacity` |
| MySQL InnoDB Details | InnoDB Checkpoint Age (age + max‑age series) | `… or (cur_lsn − ckpt_lsn)` / `… or redo_log_capacity` |
| MySQL InnoDB Details | Redo Generation Rate (×2) | `… or rate(innodb_redo_log_current_lsn)` |
| MySQL InnoDB Details | Log Write Amplification | denominator `… or rate(innodb_redo_log_current_lsn)` |
| MySQL InnoDB Compression Details | Total Redo Log Space, Max Log Space Used | same as above |

**Verification (instant query, under sysbench‑style load):**

| Panel query | PS 8.0 | MySQL 9.7 |
|---|---|---|
| Total Redo Log Space (orig) | 100663296 | **∅ (No data)** |
| Total Redo Log Space (**patched**) | 100663296 *(unchanged)* | **104857600 ✅** |
| Redo Generation Rate (orig) | 194121 B/s | **∅ (N/A)** |
| Redo Generation Rate (**patched**) | 194121 *(unchanged)* | **≈143000 B/s ✅** |
| Checkpoint Age max‑age (**patched**) | 83.35 MiB *(real Innodb_checkpoint_max_age)* | **100 MiB (redo_log_capacity) ✅** |

Screenshots (`d-solo`, kiosk) confirm: `InnoDB Checkpoint Age` **No data → populated**
(Uncheckpointed Bytes 3.86 MiB / Max Checkpoint Age 100 MiB) on 9.7 and still populated with the
real Percona value (83.35 MiB) on 8.0; `Redo Generation Rate` **N/A → 142.8 kB/s** on 9.7.

### 📋 Backlog (documented, not fixed here — become their own tickets per PMM‑15122 scope)
* **Query Cache panels** (Overview, Summary, Instances Compare) — see §5. Product decision.
* **NUMA Interleave** — `innodb_numa_interleave` removed in 9.x; no metric replacement. Hide for 9.x.
* **LRU Sub‑Chain Churn, Misc InnoDB Transactions, Transactions & Undo Records** — depend on
  Percona status vars (`Innodb_buffer_pool_pages_made_young`, `Innodb_purge_trx_id`,
  `Innodb_lsn_current`); will populate on real Percona Server 9.7. Verify when PS 9.7 ships.
* **Table Details → Top Tables by Rows Read / Rows Changed** — driven by Percona `userstat`
  (`information_schema.TABLE_STATISTICS`). Absent on upstream 9.7; present on PS 9.7 with
  `userstat=ON`. Optional enhancement: add a `performance_schema.table_io_waits_summary_by_table`
  fallback so the panels also work on non‑Percona MySQL.
* **Service Summary — `Cannot execute mysqldump … in PATH`** — this originates in
  `pt-mysql-summary` (Percona Toolkit) shipped in **pmm‑client**, not in this repo. Packaging fix:
  ensure a 9.x‑compatible `mysqldump`/`mysqlsh` is on the pmm‑client `PATH`. Client‑packaging ticket.
* "Format‑fix" items in the ticket (Buffer Pool Size of Total RAM, CPU Core Usage for
  (Un)compression, Perf‑Schema Events, Compression panels) render empty on **both** 8.0 and 9.7 in
  this environment → they need compression/NUMA/perf‑schema features enabled and load, not a 9.7
  code change. Re‑validate on a feature‑complete PS 9.7.

## 5. Query Cache — definitive finding (answers the ticket's ⚠️ note)

MySQL **removed the Query Cache entirely in 8.0**. Confirmed empirically: on **both** PS 8.0.46
and MySQL 9.7.1, `have_query_cache = NO` and neither `mysql_global_status_qcache_*` nor
`mysql_global_variables_query_cache_size` exist. Therefore the Query Cache panels
(*Top MySQL Used Query Cache, MySQL Used/Size, Query Cache Memory/Activity*) show **No data on
every PMM‑3‑supported MySQL version**, not just 9.7 — they cannot be "fixed" by re‑querying.

**Recommendation:** hide these panels for MySQL ≥ 8.0 (e.g. behind the existing `mysql_version_info`
version variable) or replace them with an explanatory "Not available in MySQL 8.0+" text panel.
This is a product/UX decision (the ticket asks to confirm with engineering) and is intentionally
**not** auto‑applied here; it is filed as a follow‑up.

## 6. Go / No‑Go for Percona Server 9.7 support

**Conditional GO.** Core telemetry works today: exporter 0.17.2 scrapes 9.x without error,
`caching_sha2_password` auth works, QAN sources register. Before GA:
1. Land the redo/LSN/checkpoint dashboard fix (this change).
2. Make the version decision for Query Cache panels (§5).
3. Re‑validate the Percona‑only panels (userstat, checkpoint, buffer‑pool churn) on a **real
   Percona Server 9.7** once released — most "missing" metrics in §2b should return there.
4. Client‑packaging: 9.x‑compatible `mysqldump` on the pmm‑client PATH for Service Summary.

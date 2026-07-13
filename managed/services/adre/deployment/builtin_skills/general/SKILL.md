---
name: general
description: 'General workload and time-window analysis: observability-map routing,
  render panel PNGs (always embed), metrics snapshot stats, QAN top slow patterns,
  MySQL EXPLAIN for top queryids; optional otel.logs. Not for one-sentence factual
  questions.'
---

# Workload and time-window analysis (general)

## Purpose

Use this runbook when the user wants **understanding of workload, performance, or “what happened” over a time range** — for example:

- Check workload in the last 24h / last night / last week
- There was a spike; what happened?
- Analyse these graphs / dashboards / panels; what do you see?
- Anomaly detection, “something was slow”, correlate metrics for a period

The agent must use **multiple data sources** in this order:

1. **pmm-inventory** — `service_id`, `node_id`, `agent_id`, names, engine type
2. **pmm_observability_map** — scoped dashboard UID, panel IDs, live PromQL `expr` (~1–3 KB)
3. **pmm_metrics_snapshot** — Tier-1 stats on panel `expr` (percentiles, change points, anomalies); not raw matrices; **does not require render**
4. **pmm_render_grafana_panel** — best-effort panel PNGs; **embed every successful** render (minimum **four** for workload/spike/anomaly unless narrowly scoped)
5. **QAN (`pmm.metrics`)** — **always** run top slow-query overview for the same window on database services (MySQL/PostgreSQL/MongoDB)
6. **MySQL Step 3b** — EXPLAIN + index review for **top 3 `queryid`s** when QAN returned non-empty `queryid` + `schema`
7. **Optional `otel.logs`** when errors or the question warrants it

**CRITICAL — Render failures are non-blocking:** Grafana panel render (502, timeout, blank PNG, curl error) must **never** abort workload analysis. After **`pmm_observability_map`**, **always** run **`pmm_metrics_snapshot`**, **QAN**, **EXPLAIN** (MySQL), and optional **logs**, then **synthesize** — even when **`rendered 0/N`**. Do **not** skip snapshot/QAN because render failed. Do **not** claim “unable to complete analysis” solely due to render errors.

**Important:** `time_runbooks` controls Holmes **runbook / TodoWrite** style features — it does **not** gate **`pmm_mysql_explain`**. Step 3b uses the PMM Actions API and must still run for MySQL workload analysis when Step 3 succeeds with `queryid`s. **Do not run logs or final synthesis until Step 3b is done or skipped only because QAN had no `queryid` / not MySQL.** If using TodoWrite, never mark snapshot/QAN/EXPLAIN blocked by render failure.

**FORBIDDEN:** unfiltered `GET /api/v1/label/__name__/values`, full dashboard JSON to the LLM, guessing panel IDs or metric names.

---

## When NOT to use this runbook

**Do not fetch this runbook** when:

- The question is a **simple factual lookup** answerable in **one short sentence** without a multi-step investigation (e.g. “How many MySQL services are there?”, “What’s the replication lag right now?”, “Is node X up?”) — answer with the minimal tool calls instead.
- The message is **casual** (hi, thanks, ping, ok) or **off-topic**.
- The user asked for **only one specific instant metric** and no interpretation across time or systems (e.g. “current QPS” only) — unless they also asked to explain workload or a period.

If unsure: prefer **not** fetching this runbook; answer directly with tools. Fetch this runbook when the user clearly wants **period analysis**, **spike explanation**, or **multi-metric / graph interpretation**.

---

## Output rules (mandatory)

- Deliver **only** the analysis: short **Summary**, **Key findings**, **Evidence** (metrics + **embedded panel images** + QAN), optional **Logs** snippets, **Recommendations** with **exact commands** (PromQL, SQL, shell/SQL for remediation).
- **Rendered images:** For every **successful** **`pmm_render_grafana_panel`** call, embed **`![short panel title](image_url)`** (blob path from tool JSON). If render failed, state **`rendered M/N`** and continue — snapshot/QAN/EXPLAIN evidence still required in Key findings.
- **QAN (mandatory for DB workload):** For MySQL/PostgreSQL/MongoDB instance workload, **Key findings** must include the **top slow patterns** from Step 3 (fingerprint context, total time, exec count) or explicitly state the QAN query returned no rows **after a successful query**.
- **MySQL + Step 3b ran (EXPLAIN):** If you called **`pmm_mysql_explain`** / **`pmm_mysql_show_*`**, **Key findings** MUST include **Query plans / indexes** with **access type**, **key**, **rows**, scan vs index, and **Recommendations** with full **`CREATE INDEX` / `ALTER TABLE` SQL**.
- **Do not** state which runbook ran, **do not** list runbook steps, **do not** narrate progress, **do not** show checklists for the user.
- Keep the reply **concise and on point**; avoid filler and repetition.

---

## Prerequisites

- Resolve **time range** from the user (`from` / `to`, e.g. `now-24h` / `now`, or explicit timestamps). State the range once in the summary.
- **Inventory (required before render):** Call **`pmm_list_services`**, **`pmm_list_nodes`**, **`pmm_list_agents`** (or get equivalents) and collect **`service_id`**, **`node_id`**, **`agent_id`**, **`service_name`**, **`node_name`**, **`version`** (and node type when available). Pass **all** of these to **`pmm_render_grafana_panel`** overrides — missing **`agent_id`** is a common cause of blank or timed-out renders.
- **Engine + intent:** Map inventory `service_type` to observability-map **`engine`** (`mysql`, `postgresql`, `mongodb`, `valkey`, `node`). Pick one **`intent`** from the user question: `workload`, `connections`, `slow_queries`, `replication`, `innodb`, `wal`, `locks`, `latency`, `memory`, `cpu_memory`, `disk_io`, `network`, `availability`.
- **Panel IDs:** Use **`pmm_observability_map`** (`engine`, `intent`, `service_id`) — use returned **`primary.dashboard_uid`** and **`panels[].id`**. Fall back to **`pmm_list_dashboard_panels`** only when the map returns warnings or unknown intent. **Never guess** panel IDs.

---

## Tool errors, retries, and time formats

- **Grafana render** (`pmm_render_grafana_panel`): best-effort visuals only. `from` / `to` without quotes; prefer **`now-<duration>`** / **`now`**. For 12h+ windows, retry once with shorter window or RFC3339 UTC sub-window. **On 502/timeout/error:** note failure, retry once, **continue with snapshot/QAN/EXPLAIN** — never abort the investigation.
- **ClickHouse QAN** (`pmm.metrics`, **`period_start`**) and **logs** (`otel.logs`, **`Timestamp`**): use **ClickHouse DateTime** bounds — e.g. `'YYYY-MM-DD HH:MM:SS'`, or `parseDateTimeBestEffort('2026-03-22T12:00:00Z')`. Query via Holmes **`pmm-clickhouse`** (`pmm_clickhouse_query`); see that toolset's llm_instructions.
- **Metrics snapshot** (`pmm_metrics_snapshot`): run **immediately after observability map** on each relevant panel `expr`. `start` / `end` = RFC3339 UTC or Unix (not `now-24h`). Requires **ADRE enabled** on PMM.
- If a query **fails**: **diagnose**, **adjust**, **retry** until success — except render, which is retried once then skipped for remaining panels while other steps continue. Do **not** claim “no slow queries” or “no logs” when the call **failed**.
- After a successful retry, base conclusions only on **actual tool output**.

---

## How to render and embed panel images (read this if images are missing)

This is the **only** supported render path:

1. **`POST /v1/grafana/render/resolve`** via tool **`pmm_render_grafana_panel`** (not legacy GET URLs, not manual `/graph/render` links).
2. **Required tool parameters:** `dashboard_uid` (from observability map), `panel_id` (numeric id from map), `from`, `to`, plus inventory overrides: **`var_service_id`**, **`var_service_name`**, **`var_node_name`**, **`var_node_id`**, **`var_agent_id`**, **`var_version`** when available.
3. **Render sequentially** — one panel per tool call; do not batch parallel renders for workload analysis.
4. **Parse the JSON response.** On success it contains **`image_url`** and **`dashboard_url`**.
5. **Embed in the user-visible answer** for each successful render:

   ```markdown
   ![MySQL Connections during incident](/v1/grafana/render/blob/abc123....png)

   [Open in Grafana](https://your-pmm-host/graph/d/mysql-instance-summary/...?from=...&to=...)
   ```

   Rules:
   - Use **`image_url` exactly as returned** (relative blob path starting with `/v1/grafana/render/blob/`). The PMM chat UI resolves it; do not prepend `PMM_URL` unless your environment explicitly requires it.
   - Use **`dashboard_url` exactly as returned** for the Grafana link line.
   - **Caption** = short panel title (from observability map `panels[].title`).
   - Place images in **Key findings** or a dedicated **Evidence (panels)** subsection — **not** only in structured report JSON.

6. **If render fails:** retry once with cleaned overrides (drop broken `var_node_name` internal FQDNs; ensure `agent_id` is set; shorten time range). If still failing, state **`rendered M/N`**, panel id/title, and error — **immediately continue** with **`pmm_metrics_snapshot`**, **QAN**, **EXPLAIN**, and other panels. Never stop the investigation because render failed.

7. **Minimum for workload/spike/anomaly:** render **at least four** panels across categories (e.g. connections, throughput/handlers, slow queries or latency, CPU or disk I/O) using ids from **`pmm_observability_map`** for intent `workload` unless the question is narrowly scoped to one metric.

---

## Steps

### 1. Observability map + metrics snapshot (scoped metrics)

**Do not** list all metric names via `__name__/values`.

1. Call **`pmm_observability_map`** with **`engine`**, **`intent`**, and **`service_id`** from inventory.
2. Note **`primary.dashboard_uid`**, each **`panels[]`** `{id, title, expr}`, **`fallback.metric_prefix`**, and any **`warnings`**.
3. For **each relevant panel `expr`** from the map (connections, handlers/QPS, slow queries, CPU, disk I/O — at least four categories for general workload), call **`pmm_metrics_snapshot`** with:
   - `query` = panel `expr` (tighten with `service_id` label if the expr is broad)
   - `start` / `end` = same logical window as Grafana (RFC3339 or Unix)
   - `max_series` = 5 (default)
4. Use snapshot **`stats`** (min, max, mean, median, p25, p75, **p95**, **p99**), **`change_points`**, and **`anomalies`** in findings — cite **real numbers** from the JSON.
5. **Scoped fallbacks only** if map warns or expr missing: **`pmm_discover_series_labels`** or **`pmm_list_metric_names`** (prefix from map fallback, max 50 names).
6. **`execute_prometheus_range_query`** — at most **one** per turn, **last resort** when snapshot is insufficient.

### 2. Grafana — render panels and embed images (best-effort visuals)

Run **after** Step 1 snapshot (or in parallel once expr is known). Render failure does **not** block Steps 3–5.

1. Using **`primary.dashboard_uid`** and **`panels[].id`** from Step 1, call **`pmm_render_grafana_panel`** for **each** panel you will show (minimum **four** for general workload/spike unless narrowly scoped).
2. Pass the **full inventory override set** (see **How to render and embed panel images**).
3. **`from` / `to`:** match the user’s analysis window (`now-24h` / `now` is fine for most cases).
4. **In the final answer:** embed **every** successful render with **`![title](image_url)`** plus **`[Open in Grafana](dashboard_url)`** — see embedding rules above.
5. Briefly interpret each panel (trend, spike, flatline, saturation) and tie to snapshot stats where possible.
6. **Panel choice** must match what you describe (do not call a connections panel “QPS”).

### 3. QAN — top slow patterns (mandatory for database services)

**Always run** for MySQL/PostgreSQL/MongoDB workload/spike questions (skip only for pure node/host questions with no DB scope).

- Query **`pmm.metrics`** for the **same time window** as Grafana, using **ClickHouse DateTime** on **`period_start`**.
- Filter by **`service_id`** (UUID from inventory). **Never** `WHERE schema = 'mysql'` to mean “this instance”.
- Aggregate by **`fingerprint`** and **`schema`**; include **`any(queryid) AS queryid`**, **`SUM(m_query_time_sum) AS total_query_time`**, execution counts, rows examined, full-scan flags when present.
- **`ORDER BY total_query_time DESC LIMIT 3`** (or top 5 in findings if more context helps).
- **Never filter on `example`**; use **`fingerprint`** for drill-down.
- If the query succeeds with **no rows**, say so briefly. If it **errors**, fix and retry — do not skip silently.

#### “Top slow patterns” query shape

```sql
SELECT
  fingerprint,
  any(queryid) AS queryid,
  schema,
  any(example) AS example,
  SUM(m_query_time_sum) AS total_query_time,
  SUM(m_query_time_sum) / COUNT() AS avg_query_time,
  COUNT() AS exec_count,
  SUM(m_rows_examined_sum) AS rows_examined,
  SUM(m_full_scan_sum) AS full_scans
FROM pmm.metrics
WHERE service_id = '<service-uuid>'
  AND period_start >= '<window-start-CH-datetime>'
  AND period_start <= '<window-end-CH-datetime>'
GROUP BY fingerprint, schema
ORDER BY total_query_time DESC
LIMIT 3
```

### 3b. Deep slow-query drill-down (MySQL) — mandatory when QAN yields `queryid`

Run **immediately after** Step 3 when the service is **MySQL** and top rows have **`queryid`** + **`schema`**. **Do not** open Step 4 (logs) before Step 3b unless skipped (not MySQL, no rows, empty `queryid`).

For **each** of the top 3 `(queryid, schema)` pairs:

1. **`pmm_mysql_explain`** or **`pmm_mysql_explain_json`**: `service_id`, `query_id` = **`queryid`**, `database` = **`schema`** only when from QAN.
2. **`pmm_mysql_show_create_table`** and **`pmm_mysql_show_index`** for tables in the plan.
3. Optionally **`pmm_mysql_show_table_status`** when size matters.
4. Tie to workload narrative; give concrete **`ALTER TABLE` / `CREATE INDEX`** SQL.

**Do not** pass fingerprint or `example` as `query_id`.

### 4. Logs — `otel.logs` (optional)

- Run when metrics, QAN, or the user point to errors/crashes, or spikes correlate with log volume.
- Use ClickHouse DateTime on **`Timestamp`**; filter service/node via **ResourceAttributes** per Holmes log rules.
- Quote **short** chronological excerpts only.

### 5. Synthesis — summary, findings, recommendations

- **Summary:** 2–4 sentences — what happened, main driver if identifiable.
- **Key findings:** bullets with **embedded panel images**, snapshot stats, **top QAN patterns**, EXPLAIN/index results (if Step 3b ran), logs (if used).
- **Recommendations:** actionable, with **exact** SQL, PromQL, ALTER, config, or shell commands.

---

## Quality checks

- Time range is **consistent** across Grafana render, snapshot, QAN, and logs — with **correct format per system**.
- **`pmm_observability_map`** ran before ad-hoc PromQL; no global `__name__` browse.
- **At least four panel images embedded** when renders succeed (unless narrowly scoped). If **`rendered 0/N`**, Key findings must still include **snapshot stats + QAN + EXPLAIN**.
- **QAN top slow patterns** queried for DB services; findings mention top offenders or explicit empty result.
- **Multiple metrics** considered via snapshot and panels, not a single series.
- **Panel titles** match narrative.
- **MySQL Step 3b** runs when QAN returned `queryid`s; final answer includes plan/index evidence.
- **Failed** tool calls retried; no false “empty” narratives on errors.
- Output contains **no** skill meta-narration.

---

## Relationship to other runbooks

- **This runbook** is the general “period + observability map + rendered panels + snapshot stats + QAN + optional logs” path, with **embedded Step 3b** for MySQL.
- Use **`alert-triggered-slow-query-analysis`** when the user wants standalone slow-query analysis from a single alert `queryid` without full period workload context.
- Skills do **not** chain automatically; select another skill from the catalog when needed.

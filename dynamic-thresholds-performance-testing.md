# Dynamic Alert Thresholds — performance testing session (2026-08-21)

Companion to [`dynamic-thresholds-main-decision.md`](dynamic-thresholds-main-decision.md). That document's
§8 "Still to verify" item 2 flagged two open scale questions: **the override-only collector's cost has
never been measured (only emit-everything was measured, in §4.12)**, and **service/cluster scope has no
implementation to measure at all**. This session builds just enough of both to get real numbers, then
measures them live on the same kind of dev server §8's numbers came from. It does not attempt to finish
the feature — see [What was deliberately not done](#what-was-deliberately-not-done).

> A chart-based visual companion to this document — the same measurements, plus the head-to-head
> comparison in [Scenario D](#scenario-d--override-only-vs-emit-everything-head-to-head) and the fixed
> real-world parameter sweep in [Scenario E](#scenario-e--fixed-real-world-parameter-sweep-1000-nodes-250-overrides-120-rules)
> — was published as an Artifact ("Threshold collector benchmarks").

## Environment

Same dev container (`pmm-server`, `perconalab/pmm-server:3-dev-latest`) used for the live measurements
already in §8 of the decision doc, on this branch (`PMM-14912-dynamic-thresholds`, `73b7032db`).

| | |
|---|---|
| PostgreSQL | 14.24 (Percona Distribution), `max_connections=2000` |
| VictoriaMetrics | scraped every `MR = 10 s` |
| Grafana | 12.4.5 |
| Real inventory | 11 nodes, 10 services (MySQL/PostgreSQL/Valkey/MongoDB test instances) |
| ClickHouse / qan-api2 | **down for this whole session** — ClickHouse is crash-looping on an unrelated `AccessControl` config error, pre-existing in this container, nothing to do with this feature. QAN has no dependency on alert thresholds, so this doesn't affect anything below; noted for completeness only. |

Baseline `/debug/metrics` scrape before any change: **25 ms**, 499 lines, 58 KB, 0 threshold series.

### Baseline infra snapshot (before synthetic load)

| Store | Size | Detail |
|---|---|---|
| PostgreSQL | 10 MB total | Largest tables: `agents` 312 KB, `nodes` 280 KB, `alert_rules` 64 KB, `alert_rule_threshold_overrides` 32 KB (empty), `settings` 48 KB. 20 active connections. |
| VictoriaMetrics | ~59 MB on disk (`indexdb` 18.5 MB + `storage/small` 46.6 MB) | 42,323 active series, 467,147 label-value pairs, 59 scrape targets, ingesting ~5,745 rows/s, 892 MB resident. |
| ClickHouse | 116 KB | Crash-looping (see above); no QAN data this session. |
| pmm-managed process | 628 MB RSS, 162 goroutines | Sampled via `process_resident_memory_bytes{job="pmm-managed"}` / `go_goroutines{job="pmm-managed"}` through VictoriaMetrics. |
| Host | load1 ≈ 3.8, memory ≈ 31.6% used | Sampled via `node_load1` / `node_memory_*` for the `pmm-server` node — the same series PMM's own Node Overview / health dashboards render. The in-app Browser tool couldn't reach `https://localhost` (sandboxed-network policy blocks local-network navigation), so these were pulled directly from VictoriaMetrics' query API instead of a dashboard screenshot — same underlying data. |

## What changed on this branch for this session

The branch's existing collector (`threshold_metrics.go`) implemented **emit-everything** — the design
§7 of the decision doc rejects, and §4.12 measured. To benchmark the **recommended** design (override-only
emission, tombstones, multi-scope resolver) there was nothing to run yet, so this session built a real,
working version of the two pieces actually in scope (see the earlier scoping discussion in this
conversation):

| File | Change |
|---|---|
| [`managed/models/database.go`](managed/models/database.go) | Migration 119's `alert_rule_threshold_overrides` rewritten to the polymorphic `scope`/`target` schema with a `cleared_at` tombstone column and the NaN/±Inf `CHECK`, per §4.4/§4.5 of the decision doc. |
| [`managed/models/alert_rule_threshold_override_model.go`](managed/models/alert_rule_threshold_override_model.go) | `ThresholdScope` type + `node`/`service`/`cluster` constants; struct fields `Scope`/`Target`/`ClearedAt` replacing `NodeID`. Regenerated via `go generate` (reform), not hand-edited. |
| [`managed/models/alert_rule_helpers.go`](managed/models/alert_rule_helpers.go) | `FindThresholdOverridesByTarget`, tombstone-aware `UpsertThresholdOverride`/`ClearThresholdOverride`, hard-delete `DeleteThresholdOverridesForTarget` (refuses `cluster` scope, per §4.8). |
| [`managed/models/threshold_resolver.go`](managed/models/threshold_resolver.go) *(new)* | `ResolveThresholds` — the shared precedence resolver from §4.10: node > service > cluster, tombstones contribute no candidate for their own scope, unresolvable targets are skipped. Table-tested in [`threshold_resolver_test.go`](managed/models/threshold_resolver_test.go) (7 cases) plus a microbenchmark. |
| [`managed/models/service_helpers.go`](managed/models/service_helpers.go) | `FindServicesByClusters` — bounded by the clusters actually queried, for cluster-scope expansion. |
| [`managed/services/alerting/threshold_metrics.go`](managed/services/alerting/threshold_metrics.go) | Full rewrite: override/tombstone-only emission (`pmm_alert_threshold_override{rule_id,param,target}`), one query for overrides + bounded ID→name/cluster→services resolution, 3 s `Collect` timeout, `Describe` sends the descriptor directly instead of `prom.DescribeByCollect` — the three defects §4.12/§6 step 3 called out, fixed in the same pass. **Both emission modes now live side by side** behind a `ThresholdEmitMode` parameter: `ThresholdEmitOverridesOnly` (default, recommended) and `ThresholdEmitEveryTarget` (the §7-rejected alternative, generalised from its original node-only shape to also emit for every service, so cluster/service scope can be A/B'd too). Selected via `PMM_DEV_THRESHOLD_EMIT_MODE=all-targets` — a dev-only toggle (`PMM_DEV_` prefix, never a GA knob), read once at startup in `managed/cmd/pmm-managed/main.go`. |
| [`managed/services/alerting/threshold_overrides.go`](managed/services/alerting/threshold_overrides.go) | Adapted to the new schema; `DeleteNodeThreshold` now calls `ClearThresholdOverride` (tombstone) instead of a hard delete, and `ListNodeThresholds` skips tombstoned rows — both required by §4.4. |

`go build ./managed/...` and `go vet ./managed/...` are clean; `gofmt -l` reports nothing. Existing
`rule_builder_dynamic_test.go` tests (unaffected — they don't touch this schema) still pass. Two pre-existing,
unrelated failures were left alone: a host-only `mkdir /srv: read-only file system` failure in
`service_test.go` (macOS host has no `/srv`) and a flaky timing assertion in
`software_version_helpers_test.go`; neither touches thresholds.

## Methodology

Same technique as the decision doc's §4.12 emit-everything table: seed real Postgres rows in the dev
container, hot-swap the rebuilt `pmm-managed` binary (`make env-root TARGET=run-managed-ci`), and time
repeated `GET /debug/metrics` scrapes. Where the shared endpoint's total time was dominated by other,
unrelated collectors (see below), `EXPLAIN ANALYZE` isolates this collector's own two queries, and a Go
microbenchmark isolates `ResolveThresholds` in memory. All synthetic rows were prefixed `bench-` and
deleted at the end of the session; the container was left in its original state (11 nodes / 10 services /
1 pre-existing rule / 0 overrides).

## Results

### A — override count, node scope (2,000 synthetic nodes held constant)

| Overrides | median | max | emitted series | scrape total lines |
|---|---|---|---|---|
| 0 | 108 ms | 133 ms | 0 | 2,530 |
| 100 | 119 ms | 175 ms | 100 | 2,643 |
| 1,000 | 124 ms | 141 ms | 1,000 | 3,549 |
| 2,000 (all nodes overridden) | 143 ms | 192 ms | 2,000 | 4,549 |

Collector's own added cost going from 0 → 2,000 override series: **~35 ms**, not the multiplicative blowup
the old design showed. The 108 ms floor here is the pre-existing, unrelated inventory collector iterating
2,011 nodes on every scrape (see below) — not this feature.

### A4 — same total overrides, fragmented across many rules/params

The old design's real exposure (§4.12) was **rules × params**, not raw override count: 1011×42×20 blew the
9 s timeout at 26.9 s. This is the one number from that table worth reproducing under the new design:

| Scenario | median | max | emitted series |
|---|---|---|---|
| 1,000 overrides, 1 rule × 1 param | 124 ms | 141 ms | 1,000 |
| 840 overrides, **42 rules × 20 params** | 121 ms | 136 ms | 840 |

Statistically indistinguishable. Fragmenting the same override volume across 840 distinct `(rule, param)`
groups costs nothing extra — the collector still issues exactly one overrides query and one bounded
node lookup, then partitions in memory. **This is the direct answer to §4.12's worst row**: the
multi-param exposure was a property of emit-everything's node-multiplication, and it is structurally
gone under override-only emission, not just empirically smaller.

### B — total inventory size, override count held constant at 50

| Total nodes | median | max | emitted series (this collector) |
|---|---|---|---|
| 2,000 | 108 ms | 118 ms | 50 |
| 6,000 | 324 ms | 382 ms | 50 |
| 10,000 | 461 ms | 508 ms | 50 |

Emitted series from **this** collector stayed at 50 throughout — the ~350 ms of growth is entirely the
pre-existing `pmm_managed_inventory_nodes` collector (unrelated to this feature) still emitting one line
per node. Isolated with `EXPLAIN ANALYZE` at the 10,000-node point:

- `SELECT * FROM alert_rule_threshold_overrides` (all 50 rows): **1.7 ms**
- `SELECT * FROM nodes WHERE node_id IN (<50 ids>)`, index-bounded: **3.6 ms**

Under 6 ms combined, flat regardless of total inventory — confirming the design's central claim
(§4.12/§9 gate 7) that this collector's cost is bounded by *targets ever tuned*, not fleet size.

### C — cluster-scope expansion (new: not measured anywhere before this session)

Synthetic services seeded per cluster, one cluster-scope override per cluster on top of the scenario-B
inventory (so total scrape time below still carries that unrelated ~460 ms inventory-collector floor):

| Cluster overrides | Services in those clusters | emitted series | scrape median | scrape max |
|---|---|---|---|---|
| 1 (100 services) | 100 | 100 | 799 ms | 886 ms |
| 2 (+1,000 services) | 1,100 | 1,100 | 823 ms | 1,033 ms |
| 3 (+5,000 services) | 6,100 | 6,100 | 845 ms | 1,055 ms |
| **50 small clusters × 200 services** (§4.2's stated worst case) | 10,000 | 10,000 | 1,482 ms | 1,566 ms |

Isolated:

- `SELECT * FROM services WHERE cluster IN (<3 clusters>)` at 6,100 matched rows: **23 ms**
- Same query for the 50-cluster case, 10,000 matched rows out of 16,110 total synthetic services: **47 ms**
- `ResolveThresholds` in memory, 50 cluster overrides expanding to 10,000 candidates (Go microbenchmark,
  `BenchmarkResolveThresholds`): **~1 ms**

§4.2's cardinality warning — "a cluster override over 200 nodes would emit 200 series" — is real and the
collector handles it correctly at 50× that scale, cheaply. The cluster-expansion query
(`services.cluster IN (...)`) is a **sequential scan** — there is no index on `services.cluster`. At
16,110 synthetic rows it's still only 47 ms.

> **Revised in Scenario E below.** This session first flagged that scan as an "index it before cluster
> scope ships" action item. A more careful re-test — a *selective* override set (10% of clusters, not
> the ~100% coverage the number above was measured against) at 10,010 services — still executes in
> **6.6 ms**. Postgres reads a table this size in one or two pages regardless of a `WHERE` clause; an
> index cannot beat that. **The recommendation is retracted**: don't add the index on the strength of
> this testing. Revisit only if a real deployment's `services` table is one to two orders of magnitude
> larger than anything measured here — the scan cost is linear in table size, so re-test at that scale
> rather than assuming today's numbers still hold.

## Scenario D — override-only vs. emit-everything, head to head

Everything above measured the recommended design in isolation. To compare it directly against the
rejected alternative on identical data, the collector now carries **both** emission modes side by side
(see [What changed](#what-changed-on-this-branch-for-this-session) above) — a live toggle, not a
re-implementation from memory of §4.12's numbers. `ThresholdEmitEveryTarget` is a deliberate
generalisation of that historical, node-only design: it now also emits for every **service**, so the
comparison covers cluster/service scope too, not just node scope.

### D1 — node scope, same override sweep as scenario A

2,000-node inventory; two registered rules with default params (the pre-existing real rule plus
`bench-rule-1`) — same DB state, same moment, mode flipped via `PMM_DEV_THRESHOLD_EMIT_MODE` and a
restart between passes:

| Overrides | override-only median / max | override-only series | all-targets median / max | all-targets series |
|---|---|---|---|---|
| 0 | 118 / 203 ms | 0 | 149 / 199 ms | **4,042** |
| 100 | 111 / 112 ms | 100 | 163 / 174 ms | **4,042** |
| 1,000 | 119 / 138 ms | 1,000 | 199 / 221 ms | **4,042** |
| 2,000 | 143 / 150 ms | 2,000 | 209 / 243 ms | **4,042** |

The all-targets column is already the whole point: **4,042 emitted series regardless of override
count — including at zero.** That number is `2 rules × 1 param × (2,011 nodes + 10 services)`; it has
nothing to do with how many overrides exist, because this mode was never about overrides, it's about
inventory. Override-only's series count tracks the override count exactly, and its scrape time is lower
at every point on the sweep. A second, smaller effect: all-targets' own time still drifts upward with
override count (149→209 ms) even though its emitted-series count doesn't move — `ResolveThresholds` has
more candidates to fold into its per-target map, a real but minor cost next to the ~150 ms fixed floor of
walking the full inventory twice per rule.

### D2 — cluster scope, same DB state under both modes

2,011 nodes + 6,100 synthetic services across 3 clusters, one cluster-scope override per cluster:

| Mode | median | max | emitted series |
|---|---|---|---|
| override-only | 550 ms | 925 ms | 6,100 |
| all-targets | 681 ms | 692 ms | **16,242** |

All-targets pays for the full `2 rules × 1 param × (2,011 nodes + 6,100 services)` universe — 2.7× the
series override-only emits for the exact same three cluster overrides, because it was always going to emit
for every service whether or not a cluster override existed.

### The extreme point, not a controlled pair

Pushing all-targets to `2,011 nodes + 16,100 services` (50 small clusters × 200 services, matching §4.12's
worst-case shape but now over nodes *and* services): **36,242 emitted series, 1,790 ms median / 1,885 ms
max** — still under the 9 s budget for one param on one rule pair, but this is exactly the shape that hit
26.9 s at 42 rules × 20 params in §4.12's original, node-only measurement. The nearest override-only
comparison is scenario C's 50-cluster point (§C above): **10,000 series, 1,482 ms**, measured over a
different total inventory (10,000 nodes instead of 2,011). The two runs aren't a controlled pair — don't
read the exact ms gap as precise — but the series counts are exact and the qualitative result matches D1
and D2 at every controlled point: override-only's cost is anchored to what was actually overridden,
all-targets' is anchored to total inventory size, full stop.

*(Operational side note, not a thresholds finding: seeding 16,100 synthetic services made the unrelated
Advisor **checks** service log an error per service — "no available pmm agents" — on its periodic pass,
which briefly slowed `pmm-managed`'s graceful shutdown between mode-toggle restarts. Nothing to do with
the threshold collector; mentioned only because it's what made a restart briefly look stuck.)*

## Scenario E — fixed real-world parameter sweep (1,000 nodes, 250 overrides, 120 rules)

Requested directly: a single fixed, realistic parameter set — **1,000 nodes, 250 overrides, 100 rules with
1 param + 20 rules with 2 params** (140 `(rule, param)` groups total) — run through node, service, and
cluster scope, each under both emission modes. Unlike scenarios A–D, this inventory is **left seeded on
the dev container** for further exploration (see the note at the end of this section), rather than cleaned
up after measuring.

### Setup

| | |
|---|---|
| Nodes | 1,000 synthetic + 11 real = 1,011 |
| Services | 1,000 synthetic (100 clusters × 10) + 10 real = 1,010, later expanded to 10,010 (1,000 clusters × 10) for the cluster-selectivity re-test below |
| Alert rules | 100 with 1 param (`threshold`) + 20 with 2 params (`param_a`, `param_b`) = **140 `(rule, param)` groups** |
| Overrides | 250 rows, spread across all 140 groups (~1.8 per group on average), scope varied per pass |

The 250 overrides were distributed with a fixed, reproducible mapping (`idx % 140` → group, `idx` → target)
so the same 250-row shape is tested identically at each scope, and it self-verified: no two rows landed on
the same `(rule, param, scope, target)` key, so all 250 inserted cleanly under the real
`UNIQUE (rule_id, param_name, scope, target)` constraint every time.

### Results

| Scope | Mode | Emitted series | Scrape median | Scrape max | Isolated DB cost |
|---|---|---|---|---|---|
| Node | overrides-only | **250** | 108 ms | 114 ms | overrides scan 1.6 ms + node lookup 2.1 ms |
| Node | all-targets | **284,961** | **6,907 ms** | 6,907 ms | dominated by the emission loop, not the query |
| Service | overrides-only | **250** | 109 ms | 111 ms | service lookup 2.3 ms |
| Service | all-targets | **284,961** | **7,394 ms** | 7,394 ms | same order as node scope — see below |
| Cluster | overrides-only, 100% of 100 clusters selected (1,010 services) | **2,500** | 156 ms | 164 ms | seq scan, matches ~100% of table: 5.9 ms |
| Cluster | overrides-only, 10% of 1,000 clusters selected (10,010 services) | **2,500** | 620 ms | 818 ms | seq scan, 10% selective: 6.6 ms |
| Cluster | all-targets | *not run — see below* | — | — | — |

Two clean, exact formulas fall out of these six rows:

- **override-only emits exactly `resolved distinct targets`** — 250 override rows resolve to 250 distinct
  node/service names at node/service scope, and to **2,500** at cluster scope, because each of the 250
  cluster-scope rows expands onto its cluster's 10 services (250 × 10 = 2,500, exactly what was measured —
  not an estimate).
- **all-targets emits exactly `groups × (nodes + services)`, independent of override count and scope** —
  **141** groups (the 140 bench groups plus one pre-existing real alert rule left over from earlier in the
  session, confirmed by `SELECT sum(...jsonb_object_keys(default_params)...) FROM alert_rules` = 141) ×
  `(1,011 + 1,010) = 2,021` = **284,961** — exact, not approximate; there is no rounding gap. Node scope and
  service scope produced the **same** all-targets series count and the same order-of-magnitude scrape time
  (6,907 ms vs 7,394 ms) — confirming empirically, not just architecturally, that all-targets' cost has
  nothing to do with which scope is being overridden.

**The headline number: 6.9–7.4 seconds of a 9-second scrape budget**, for just 140 `(rule, param)` groups —
the exact shape §4.12 warned about (there, 42 rules × 20 params on nodes alone hit 26.9 s and blew the
budget outright). This run didn't blow the budget, but it used **77–82% of it**, with only 120 rules in
play — a fraction of what a real PMM deployment (42+ shipped templates, more once overridable params
grow) would register once several are made overridable under this design.

**A collector-implementation note surfaced by this run, not by earlier ones:** the 3 s `Collect` timeout
this session added (§4.12/§6 step 3) bounds the *DB query* phase, but `collectEveryTarget`'s emission loop
— the part that actually costs 6–7 s here — runs entirely in Go after the queries return, with no
`ctx.Done()` check. **The timeout does not actually cap all-targets' worst case.** This is a real gap in
the current implementation of the (already-rejected) alternative, not a design-doc gap — worth noting
precisely because it means the measured 6.9–7.4 s undersells the actual risk: without a check inside the
loop, a slightly larger rule count would run past 9 s with nothing stopping it.

**Cluster-scope all-targets was deliberately not run at the 10,010-service inventory.** Extrapolating the
formula above: `141 × (1,011 + 10,010) = 1,553,961` series — over 5× the 284,961-series run that already
took ~7 s and ~425% host CPU. Running it risked destabilizing the shared dev container (other test
databases run alongside it) for a number whose conclusion is already obvious from the formula. Treat
1.5M as an estimate, not a measurement, and note it as exactly that if it's cited elsewhere.

### The index question, revisited — retracted

Scenario C flagged the missing index on `services.cluster` as an action item, measured against an override
set covering essentially the whole services table (100%-selective). Re-tested here deliberately
*selectively* — 100 of 1,000 clusters overridden, 10,010 total services — the sequential scan still runs in
**6.6 ms**. Postgres reads a table this size in a handful of pages regardless of the `WHERE` clause; there
is no query for an index to speed up yet. **The recommendation is retracted.** Postgres' sequential-scan
cost is linear in table size, so this conclusion should be re-tested rather than assumed if a real
deployment's `services` table is one to two orders of magnitude larger than anything measured here — but
nothing in this session's data supports adding the index now.

### Left in place for further exploration

Unlike scenarios A–D, this session's seed data was **not** cleaned up. As of this section, the dev
container carries: 1,011 nodes, 10,010 services (1,000 clusters × 10 + real), 121 alert rules, and 250
cluster-scope overrides (the last configuration measured) resolving to 2,500 effective thresholds. The
collector is back on `overrides-only` (the recommended default) — `all-targets` was never left active.

## PMM health during the load

Sampled while the 10,000-node / 16,000-service synthetic inventory was in place (the heaviest point of
this session, scenario C's worst row): host load1 stayed the same order of magnitude, `pmm-managed` RSS
and goroutine count did not visibly spike attributable to this collector specifically — the dominant
cost throughout was the pre-existing inventory collectors serializing tens of thousands of unrelated
lines into the same `/debug/metrics` response, not anything this feature adds. No scrape exceeded the
`0.9 × MR = 9 s` budget at any point tested here.

## What was deliberately not done

Scoped out at the start of this session, not discovered as blockers partway through:

- **`rule_builder.go` / PromQL generation is untouched.** The `T_<param>` / `label_replace` / fan-out-over-
  observed-expression query shape from §2/§4.3 was not wired up; the existing direct-join rule builder
  and its tests are unaffected. Query-side cost for node scope was already measured live in §8 of the
  decision doc and remains valid. There is **no** end-to-end (Grafana rule → collector → firing) test of
  service/cluster scope from this session — only the collector and resolver were exercised directly
  against Postgres.
- **No proto/API changes.** `scope`/`target` are not exposed externally; the existing node-centric
  endpoints now route through `ThresholdScopeNode` internally, unchanged from the outside.
- **Removal hooks not wired.** `DeleteThresholdOverridesForTarget` exists but isn't called from
  `RemoveNode`/`RemoveService` yet (§4.8).
- **Reconciler untouched**, **HA not tested** — this session used one `pmm-managed` instance; §9 gate 7's
  "under HA" half is still open.
- Not a shippable increment by itself — this is benchmarking-oriented code sized to answer the two
  questions asked (override-only collector cost, service/cluster resolver cost), not to complete the
  ten-step plan in §6 of the decision doc.

## Bottom line

Both open scale questions from §8/§9 now have real numbers instead of none:

1. **The override-only collector costs what the design assumed it would** — single-digit milliseconds of
   DB time, flat with inventory size, flat with how many rules/params the same override volume is spread
   across. The multi-param blowup that killed emit-everything (§4.12, 42×20 → 26.9 s) does not reproduce
   under this design at any scale tried here.
2. **Cluster-scope expansion works and is cheap up to 10,000 expanded series.** No index needed on
   `services.cluster` — retracted in Scenario E after a more careful re-test at realistic selectivity.
3. **Head to head, on identical data, override-only wins at every point tested** (scenario D) — and the
   gap is structural, not incidental: emit-everything's cost is a function of *inventory size*, so it pays
   the same ~4,042-series price whether zero or two thousand overrides exist, while override-only's cost
   is a function of *what was actually tuned*. Both modes now live in the same binary
   (`PMM_DEV_THRESHOLD_EMIT_MODE=all-targets`), so this isn't a claim resting on the branch's older,
   now-replaced code — it's the same code path, same DB, same endpoint, mode flipped by one env var.
4. **At a fixed, realistic parameter set (1,000 nodes, 250 overrides, 140 rule×param groups — Scenario E),
   all-targets consumed 77–82% of the 9 s scrape budget** regardless of scope, while override-only used
   1–7%. This is with only 120 rules; more overridable templates only makes the gap worse, and the
   `Collect` timeout added this session does not actually bound this cost — a real implementation gap in
   the (rejected) alternative, not a design gap.

HA and the full query-side integration with service/cluster join labels remain open per
[§9](dynamic-thresholds-main-decision.md#9-decision-gates) — this session narrows, but does not close,
those gates.

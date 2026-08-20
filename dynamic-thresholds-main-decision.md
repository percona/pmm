# Dynamic Alert Thresholds — decision and implementation plan (against `main`)

Per-node (and later per-service, per-cluster) threshold overrides for alert rules created from
templates, specified against **`main` at `a807f56d8`**. Every file and line reference below was checked
against `main`, not against any feature branch.

**Scope of this document**

| In scope | Out of scope |
|---|---|
| Migration, reform models, CRUD helpers | UI (thresholds modal, hooks, Grafana-side trigger) |
| Threshold metric collector | Dedicated metrics endpoint + own scrape job (costed as a follow-up) |
| Rule builder: threshold-query injection and single-expression desugaring | Grafana `UpdateRule`/`DeleteRule` RPCs |
| gRPC/REST API for reading and writing overrides | Service and cluster **behaviour** (schema and API are generalised now; only node scope is implemented) |
| Reconciler for orphaned rule registry rows | |

Written greenfield: it assumes nothing exists beyond `main`.

> **Revision note — reviewed and revised.** This document has been through review. Two direct
> correctness contradictions were found and are now fixed in place: the "omit the threshold query when
> there are no overrides" optimisation (§5.1), which silently defeated the write-once guarantee; and a
> claim that node deletion cascades override rows (§4.7), which the polymorphic schema in §4.4 makes
> impossible. Four further findings — bounded outage tolerance, cross-store failure semantics, the
> multi-scope resolver contract, and desugaring safety — are now specified rather than implied, and the
> estimate carries its assumptions. Items that could invalidate GA behaviour have been promoted from
> "open questions" to **§9 Decision gates**.
>
> Where this document states a fact as *measured*, it was measured on a live server. Where it
> *extrapolates*, it says so. Unverified assumptions are listed in §8.

---

## 1. The problem

Alert rules created from templates bake the threshold into the query (`$A > 80`). Changing the threshold
for one node means editing or recreating the rule. We want a per-node override that is a **data change**,
not a rule change.

## 2. Recommendation in one paragraph

Emit a **VictoriaMetrics gauge carrying only the overrides**, materialise the **default at query time**
from PMM's existing inventory metric, and have the rule compare its observed query against that
combination. The Grafana rule is written **once at creation and never rewritten**, so tuning a threshold
never disturbs alert state. Postgres is the single source of truth; VictoriaMetrics is a derived
transport.

```promql
# the injected threshold step, T_<param>
max by (node_name) (
  label_replace(pmm_alert_threshold_override{rule_id="<uuid>", param="<name>"},
                "node_name", "$1", "target", "(.*)")
)
or
(max by (node_name) (last_over_time(pmm_managed_inventory_nodes[15m])) * 0 + <default>)
```

## 3. What `main` already provides

This is the part that makes the estimate small. On `main` today:

| Foundation | Where |
|---|---|
| **Multi-expression templates** (`queries:` / `expressions:` / `condition:`) | `managed/pi/alert/query.go:25` (`TemplateQuery`), `:38` (`UsesMultipleExpressions`), `:55`–`:131` (validation) |
| Rule builder with a multi-expression path | `managed/services/alerting/rule_builder.go:58` (`buildGrafanaRuleData`), `:81` (`buildMultiExpressionRuleData`) |
| Prom-query and math-expression step builders | `rule_builder.go:141` (`newPromQueryData`), `:165` (`newMathExpressionData`) |
| Filter application | `rule_builder.go:120` (`fillAndFilterExpr`) |
| Rule creation flow | `managed/services/alerting/service.go:679` (`CreateRule`), annotations filled at `:744` |
| Grafana rule creation | `managed/services/grafana/client.go:712` (`CreateAlertRule`) |
| **A per-node inventory metric, already scraped** | `managed/services/inventory/inventory_metrics.go:77-80` → `pmm_managed_inventory_nodes{node_id, node_type, node_name, container_name}` |
| A per-service/agent inventory metric | `inventory_metrics.go:71-75` → `pmm_managed_inventory_agents{…, service_id, service_name, node_id, node_name, …}` |
| Collector registration + `/debug/metrics` exposition | `managed/cmd/pmm-managed/main.go:942`, `debugAddr` at `:123` |
| That endpoint already scraped by VM | `managed/services/victoriametrics/scrape_configs.go:80` (job `pmm-managed`, interval `MR` = 10 s per `managed/models/settings.go:209`, timeout `0.9 × MR` per `scrape_configs.go:41`) |
| Stable identifiers on observed series | `Node.UnifiedLabels()` `managed/models/node_model.go:118`, `Service.UnifiedLabels()` `managed/models/service_model.go:115` |
| Leader-election hook for background work | `managed/services/ha/haservice.go:569` (`AddLeaderService`), `:597` (`IsLeader` — true when HA is disabled) |
| Migrations up to **118** | `managed/models/database.go:1184` → this feature takes **119** |

What is **absent** on `main` and must be built: the `overridable` param flag, both tables, the collector,
threshold-query injection, the API, the reconciler, and (per the scope decision) single-expression
desugaring. `main` has **no** `UpdateAlertRule`, `ListAlertRules` or `DeleteAlertRule` — and the
recommended design needs only `ListAlertRules`, for the reconciler.

Template inventory on `main`: **42 templates — 38 single-expression, 4 multi-expression.**
`pmm_node_high_cpu_load` is already multi-expression, so it becomes overridable with one YAML line.

## 4. Design

### 4.1 Rule shape

Every overridable rule compiles to the same steps, whatever the template looked like:

| refId | datasource | body |
|---|---|---|
| `A`, `B`, … | Metrics (VM) | the template's observed queries, unchanged |
| `T_<param>` | Metrics (VM) | the threshold expression from §2, one per overridable param |
| `C` | `__expr__` math | the template's expression, with `[[ .param ]]` swapped for `$T_<param>` |

Written once. A threshold change touches one Postgres row and nothing in Grafana.

### 4.2 Metric shape

```
pmm_alert_threshold_override{rule_id, param, target}     # gauge
```

Exactly three labels. Value = the effective threshold for that target, in the param's native unit,
**after precedence resolution in Go**.

| Label | Why |
|---|---|
| `rule_id` | Scopes the selector so one rule cannot pick up another's series |
| `param` | Required for multi-param rules |
| `target` | The **value of the rule's join label** (`node_name` / `service_name` / cluster value), resolved in Go from the ID stored in Postgres |

Excluded deliberately: **`scope`** (precedence cannot be expressed in PromQL — §4.6), the join label by
name (its *name* varies per rule and a fixed `prom.NewDesc` cannot vary label names, so generic `target`
plus one `label_replace` keeps a **checked** collector), the default value (a literal in the rule),
and rule metadata (`template_name`, `rule_title` — they churn on rename; see §4.4 for why they are not
stored at all).

```go
desc: prom.NewDesc(
    "pmm_alert_threshold_override",
    "Effective alert threshold override for a rule parameter and target. Emitted only where an "+
        "override exists; targets without one fall back to the rule's default, which the rule "+
        "query materialises from inventory.",
    []string{"rule_id", "param", "target"},
    nil,
),
```

Implement `Describe` directly (`ch <- c.desc`) rather than via `prom.DescribeByCollect`, which would run
a full `Collect` — and therefore a database query — merely to describe the collector.

**Cardinality:** one series per `(rule, param, target-covered-by-an-override)`. Because precedence is
resolved in Go, a coarse-scope override **expands** — a cluster override over 200 nodes would emit 200
series. Bounded by "targets actually covered", which is why this never approaches `rules × params × nodes`.

### 4.3 Why each clause of the threshold query

| Clause | Job |
|---|---|
| `label_replace(…, "<join>", "$1", "target", "(.*)")` | Maps generic `target` onto whichever label this rule joins on. One fixed descriptor serves every scope. |
| `max by (<join>)` | Three jobs: **strips `instance`/`job`** (the scrape target is `pmm-server`/`pmm-managed`, which can never match an observed series' `instance`); **reduces both sides of `or` to identical label sets**, without which `or` returns both instead of preferring the left; and **collapses HA duplicates**. Measured cost: none — `samplesScanned` is identical with and without it. |
| `or` | Set union preferring the left: "override if present, else default". Requires identical label sets, which `max by` guarantees. |
| `* 0 + <default>` | Preserves the label set, replaces the value → one default series per known target. |
| `last_over_time(…[15m])` | **Fan-out only.** Keeps the default resolving across a pmm-managed restart. |

> **The window is `[15m]`, and it is a measured value.** VictoriaMetrics does **not** short-circuit
> `last_over_time`; it scans the range. Via `trace=1` on VM v1.147.0 at MR = 10 s, samples scanned per
> series: bare selector **61**, `[15m]` **91** (1.5×), `[1h]` **361** (5.9×), `[7d]` **11,246** (184×).
> Even a bare selector scans 61 because VM's `default_rollup` fetches its whole ~10-minute lookbehind, so
> an explicit `[15m]` costs only 1.5× while making the protection explicit instead of dependent on a
> server-side flag.

> **Never wrap the override series in `last_over_time`.** Clearing an override works by the series
> *ceasing to be emitted*; a window would keep the deleted value resolving for its whole length.

### 4.4 Postgres schema (migration 119)

Generalised for the confirmed roadmap. The decisive constraint: **`cluster` is a label value, not an
entity** — there is no `clusters` table, so it can never have a foreign key, and the same holds for
`environment`, `replication_set` and custom labels. Keeping FKs for node/service but not cluster would
force per-scope branching through every query and handler, so the target column is polymorphic.

```sql
CREATE TABLE alert_rules (
    rule_id          VARCHAR NOT NULL,                                    -- PMM-minted; the identity
    grafana_rule_uid VARCHAR CHECK (grafana_rule_uid <> ''),               -- cached handle, NOT identity
    params           JSONB   NOT NULL,                                    -- see below
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL,
    PRIMARY KEY (rule_id),
    UNIQUE (grafana_rule_uid)
);

CREATE TABLE alert_rule_threshold_overrides (
    id          VARCHAR NOT NULL,
    rule_id     VARCHAR NOT NULL REFERENCES alert_rules (rule_id) ON DELETE CASCADE,
    param_name  VARCHAR NOT NULL CHECK (param_name <> ''),
    scope       VARCHAR NOT NULL CHECK (scope <> ''),    -- 'node' | 'service' | 'cluster'
    target      VARCHAR NOT NULL CHECK (target <> ''),   -- node_id | service_id | cluster label value
    value       DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (rule_id, param_name, scope, target)
);

CREATE INDEX alert_rule_threshold_overrides_target_idx
    ON alert_rule_threshold_overrides (scope, target);
```

`CHECK (x <> '')` follows repo convention (55 such constraints in `database.go`). **No
`CHECK (scope IN (…))`** — validate the enum in Go so adding a scope needs no migration.

#### The `params` snapshot

Rule metadata that Grafana already holds — title, folder, rule group, template name — is **not** stored;
it is read from the rule, which is authoritative (§4.4). `params` is the one thing that is irreducible. It
is a snapshot, keyed by param name:

```json
{
  "threshold": {
    "default":    80,
    "join_label": "node_name",
    "scopes":     ["node"],
    "unit":       "%",
    "summary":    "A percentage from configured maximum",
    "min":        0,
    "max":        100
  }
}
```

Nothing else can supply it. The default *is* in the rule query, but as a PromQL literal, and recovering it
means parsing generated text — rejected in §7. Taking it from the *current* template instead lets the API
report a default the rule does not actually use, if the template was edited after the rule was created.
And it cannot live on the overrides table, because a param with **no** override has no row there.

#### This is also what makes changing a default possible later

Because the effective default is stored *and* rendered into the rule, the two can be reconciled: update
`params[p].default`, re-render the `T_<param>` step, `PUT` the rule. It is deliberately **not cheap** — a
rule-definition change resets alert state for that rule (§7) — but that is acceptable for a rare,
deliberate administrative act, unlike per-target tuning. Without the snapshot there would be nothing to
change *from*, only a literal buried in a query.

If dynamic defaults ever become routine rather than rare, the answer is not to optimise this path but to
move the default into data as well — see §10, which treats that as the trigger to revisit the
thresholds-datasource alternative.

#### How `scope` and `target` are used

`scope` and `target` are not just descriptive — together they drive three different code paths.

**Write path (API).** `scope` says which kind of thing is being targeted, `target` identifies it.
`SetNodeThreshold(node_id, …)` is the node-scope alias: it becomes `scope='node'`,
`target=<node_id>` (§4.5). Validation checks that `scope` is legal for that param per its `scopes` list,
and that `target` exists where existence is checkable. `UNIQUE (rule_id, param_name, scope, target)` allows
a node override *and* a cluster override to coexist for the same param — precedence decides which wins.

**Emit path (collector).** `scope` selects what to do with `target`:

| `scope` | `target` holds | Collector action |
|---|---|---|
| `node` | `node_id` | resolve → `node_name` via `nodes`; skip if it no longer resolves |
| `service` | `service_id` | resolve → `service_name` via `services`; skip if it no longer resolves |
| `cluster` | the `cluster` label value | no lookup — but **expand** onto the param's join label (below) |

The skip-if-unresolvable behaviour is what makes a stale row inert (§4.8), and the emitted label is always
the *resolved* value, never the stored ID, because the rule joins on names.

**Read path (API).** `(scope, target)` is the lookup key: "all thresholds for this node" is
`WHERE scope='node' AND target=$1`, which is what the `(scope, target)` index exists for. The same
precedence function then produces the effective value (§4.10).

##### Which scopes are legal for a param — a rule, not a list

A coarse scope has to be projected onto the param's join label, and that projection must be
**unambiguous**. The general rule:

> A scope `S` is legal for a param whose join label is `L` **iff the mapping `L → S` is a function** in the
> inventory — i.e. every value of `L` maps to at most one target at scope `S`.

Applying it:

| join label | scope | `L → S` | Legal? |
|---|---|---|---|
| `node_name` | `node` | `node_name → node_id` | ✅ one-to-one |
| `service_name` | `service` | `service_name → service_id` | ✅ one-to-one |
| `service_name` | `cluster` | `service_name → cluster` | ✅ each service has exactly one cluster, so a cluster override expands onto its services without conflict |
| `node_name` | `cluster` | `node_name → cluster` | ❌ a node can host services from several clusters |
| `node_name` | `service` | `node_name → service_id` | ❌ a node hosts many services, so two service overrides could claim the same node |

So a node-joined param accepts `{node}`; a service-joined param accepts `{service, cluster}`. This is a
**parse-time validation** on `override_scopes`, not a runtime tie-break — the illegal rows must never
exist. It also generalises to labels not yet supported (`environment`, `replication_set`) without new
case-by-case reasoning.

> **Consequence for the collector.** Expanding a coarse scope requires knowing the param's `join_label`,
> so the collector reads `alert_rules.params` **only once multi-scope behaviour ships**. In the node-only
> first increment every override is `scope='node'` onto `node_name`, resolution is driven by `scope`
> alone, and the collector touches only the overrides table plus inventory.

#### `grafana_rule_uid` — a cache, never the identity

`pmm_rule_id` stays the identity: PMM mints it, stamps it as a rule label, and it is stable by
construction. The Grafana UID is stored **only as a handle** for cheap direct addressing —
`GET/PUT/DELETE /api/v1/provisioning/alert-rules/{uid}` needs no folder or group, which is exactly what a
future default-change or delete path wants.

It must be treated as a cache — though **not** for the reason an earlier draft gave. That draft said
"UIDs are not stable across delete-and-recreate", citing a PoC rule that changed UID during provenance
testing. That is not a lifecycle PMM has: there is no `DeleteRule` RPC, no `DeleteAlertRule` on the
client, and this design never rewrites a rule after creation. A plain user delete-then-create is also not
the problem — the replacement carries no `pmm_rule_id` label, so the old rule is simply *gone* and its
registry row is garbage, not "the same rule with a new UID".

The real staleness paths are narrower, and all of them are **Grafana-side copy operations that preserve
labels while minting a new UID**:

- duplicating a rule in the Grafana UI;
- alert-rule export/import, including provisioning-file restore;
- Grafana backup/restore.

So the handle can go stale, which is why:

- the column is **nullable** — the value may not be known yet, and code must work without it;
- reads that matter fall back to matching on the `pmm_rule_id` label;
- a `404` on direct addressing means "refresh the handle", not "the rule is gone" — **re-resolve by label and update the stored UID**, which is self-healing and free, because the reconcile/read pass already holds both the UID and the label for every rule.

#### The same copy operations break `pmm_rule_id` uniqueness — and mostly that is fine

Duplicating a PMM rule in Grafana produces **two rules carrying the same `pmm_rule_id`**.
`UNIQUE (grafana_rule_uid)` does not help: the collision is on the identity *label*, inside Grafana, where
PMM can impose no constraint.

The important realisation is that **the coupling lives in the query text, not in the label.** The copy's
`T_<param>` step literally contains `rule_id="<original>"`, so a duplicated rule evaluates against the
original's overrides no matter what identity scheme is used. No scheme can prevent that, because copying
copies the query. Treat it as intended behaviour: **duplicated rules share thresholds.**

What must change is every place that assumed a 1:1 mapping:

| Concern | Resolution |
|---|---|
| Caching the UID | If a `pmm_rule_id` maps to more than one Grafana rule, store **`NULL`** — refuse to cache an ambiguous handle rather than picking one arbitrarily. |
| Reconciliation | It only asks *"is this `pmm_rule_id` present at all?"* — a boolean, so duplicates are harmless. Do not add "which one". |
| `ListNodeThresholds` | May legitimately return the same threshold under two rule titles. That is honest — there really are two rules — so present both rather than silently collapsing them. |
| Operator visibility | Count duplicated `pmm_rule_id`s during the reconcile pass and expose the count as a gauge (plus a log line), so this is observable instead of surprising. |

None of this requires preventing duplication, which is not preventable without read-only rules — rejected
in §7 as group-wide and irreversible.

**Capturing it costs nothing.** The Ruler API POST already returns the UIDs it created —
`{"message":"rule group updated successfully","created":["ffvj7sosfptdsd"]}` — and `CreateAlertRule`
(`managed/services/grafana/client.go:712`) currently returns only `error`, discarding that body. Have it
parse and return the created UID instead of adding a second round-trip. Note an *update* returns
`"updated"` rather than `"created"`, so handle both.

**What `target` holds:** the stable **ID** for node/service (`node_id`, `service_id` — never reused), and
the **label value** for cluster (which has no ID). The query joins on the *name*, because that is what
templates aggregate by, so the collector resolves ID → name at emit time, bounded by the override count:

```go
overrides, _ := models.FindThresholdOverrides(tx.Querier)                  // V rows, one indexed query
nodes, _     := models.FindNodesByIDs(tx.Querier, nodeIDsOf(overrides))    // WHERE node_id IN (…)
// scope='cluster' needs no lookup: target IS the label value.
```

> **In the node-only increment the collector never reads `alert_rules`.** It emits only overrides, takes
> each value from the override row, and picks the resolution table from that row's `scope` — so it needs
> neither defaults nor `params`. (Coarse-scope expansion later requires `params[p].join_label`; see "Which
> scopes are legal for a param" below.) Either way, nothing in the emit path knows whether the rule still
> exists — which is exactly why an orphaned registry row keeps producing series. See §4.9.

That resolution doubles as cleanup: an override whose target no longer resolves emits nothing, so a stale
row is inert and invisible, and GC becomes tidiness rather than correctness.

**Note on `node_name` reuse.** `UNIQUE (node_name)` (`database.go:117`) guarantees uniqueness at any
instant but **not across time** — rebuild a host, re-register with the same hostname, and you get the same
`node_name` with a fresh `node_id`. Storing `node_id` and rendering `node_name` is what keeps a stale
override from silently attaching to a different machine.

### 4.5 Value and API validation

`DOUBLE PRECISION` accepts non-finite values, and this would be **`main`'s first float column**, so there
is no existing precedent to inherit. Guard at both layers.

Database:

```sql
CHECK (value = value                            -- rejects NaN (NaN <> NaN)
       AND value > '-Infinity'::float8
       AND value < 'Infinity'::float8)
```

API, in this order, each with a defined code:

| Check | Code |
|---|---|
| `value` is finite (not NaN, not ±Inf) | `InvalidArgument` |
| `value` within the param's declared `range` | `InvalidArgument` |
| param exists on the rule | `NotFound` |
| param is `overridable` | `FailedPrecondition` |
| `scope` is a supported value, and legal for this param per `override_scopes` | `InvalidArgument` |
| `target` exists, where existence is checkable (node/service; not cluster) | `NotFound` |
| rule exists in the registry | `NotFound` |

**Alias conflict.** The API carries both `node_id` and the general `scope`/`target` pair (§6 step 7).
`node_id` is an alias for `scope='node', target=<node_id>`. If both are supplied and disagree, reject with
`InvalidArgument` — do not silently prefer one. Freeze this rule in the proto comments before the API
ships, since the fields are permanent once released.

### 4.6 Precedence is resolved in Go, not PromQL

Three measurements settle this:

| Test | Result |
|---|---|
| `or` across **different** label sets | returns **both** series — no precedence |
| `or` across **identical** label sets | left wins — so `or` precedence needs same-label-set terms |
| node override 50 + cluster override 90 under `max by (node_name)` | **90** — `max` picks the larger value, not the more specific scope |

Reducing to a common label set is required for `or` to mean "prefer the left", but that reduction is
exactly what destroys the scope information needed to rank by. So the collector emits only the effective
value per target, and **`ListNodeThresholds` and the collector must share one precedence function** —
PromQL provides no backstop if they diverge. Proposed order: `node` → `service` → `cluster`.

### 4.7 Semantics that fall out for free

- **Set an override** → row inserted → series appears within one scrape → `or` prefers it.
- **Clear an override** → row deleted → series stops → falls through to the default. **No tombstone needed.** Measured latency **280–300 s** (VM's lookbehind).
- **New node** → the fan-out picks it up; nothing to write.
- **Node or service deleted** → **there is no cascade for the target.** The only foreign key is `rule_id → alert_rules`, so deleting a *rule* cascades its overrides; deleting a *node or service* does not. The polymorphic `target` column (§4.4) cannot carry an FK, because `cluster` targets have no referent table. Two consequences, and both matter:
  - **Immediately harmless:** ID → name resolution fails for the deleted entity, so the collector emits nothing for it. The row is inert from the next scrape onward — no phantom series, no wrong threshold.
  - **But the rows must still be cleared.** Left behind, they accumulate silently, they show up in any admin/debug listing of overrides, and they make `ListNodeThresholds`-style queries return entries for entities that no longer exist. **Removing a node or a service must clear its overrides** — see the garbage-collection spec below.
- **pmm-managed down** → protection is **bounded by the fan-out window, and degrades in two steps**. An earlier draft said simply "alerting continues", which overstated it:

| Elapsed | Behaviour |
|---|---|
| 0 – ~5 min | Overrides and defaults both resolve. Normal. |
| ~5 min | Override series stop resolving → **overridden targets silently revert to their defaults**. Behaviour change #1. |
| ~5 – 15 min | Defaults still resolve via `last_over_time[15m]`. Alerting works, at defaults. |
| **> 15 min** | Fan-out empty → `T` empty → NoData → **OK**. **Alerting silently stops** while observed metrics are still flowing. |

  This makes the window a **reliability parameter, not only a cost knob** — it couples the choice in §4.3
  to the NoData decision in §9. Detection does not require new plumbing: **`up{job="pmm-managed"}`
  already exists on `main`** and covers both degradation points, so an alert on it is the day-one
  mitigation. The dedicated endpoint's `up{job="pmm-thresholds"}` would later attribute the failure more
  precisely.

### 4.8 Clearing overrides when nodes and services are removed

`target` carries no foreign key, so deletion of the referenced entity is handled **in the removal API
itself** — not by a background sweep. PMM already does dependant cleanup this way, which makes the hook
idiomatic rather than novel.

**The mechanism: delete inside the existing removal transaction.**

```go
// managed/models/node_helpers.go, in RemoveNode
DeleteThresholdOverridesForTarget(q, ThresholdScopeNode, id)

// managed/models/service_helpers.go, in RemoveService
DeleteThresholdOverridesForTarget(q, ThresholdScopeService, id)
```

Three properties of the existing code make this sufficient on its own:

1. **The DB restricts rather than cascades.** `services.node_id` is a plain `FOREIGN KEY (node_id) REFERENCES nodes (node_id)` with no `ON DELETE CASCADE`, so PMM must remove dependants explicitly — which is exactly why these chokepoints exist: `RemoveNode` (`node_helpers.go:255`), `RemoveService` (`service_helpers.go:354`), `RemoveAgent` (`agent_helpers.go:1444`), each taking a `RemoveMode`.
2. **The helpers compose.** `RemoveNode` with `RemoveCascade` finds the services on that node and calls `RemoveService(…, RemoveCascade)` for each, which in turn routes through `RemoveAgent`. So deleting a node clears its own node-scoped override **and** the service-scoped overrides of every service on it, with no second code path and no gap.
3. **It is atomic.** The delete runs in the same reform transaction as the removal, so there is no window in which overrides outlive their entity — unlike a sweep, which is eventually consistent by construction.

It also sidesteps the awkwardness a sweep would have had: no leader-election question (the delete happens
wherever the removal API is served) and no lag before the UI reflects reality.

**Retained as a free backstop, not as the mechanism:** the collector resolves `target` IDs to join-label
names and **skips anything that no longer resolves**, so a row missed by any future removal path — or
inserted by direct SQL — can never produce a wrong threshold. It is simply inert.

**Never delete `scope='cluster'` rows.** There is no "delete a cluster" operation to hook, and more
importantly a cluster override with no matching services is **dormant, not stale**: services may be added
to that cluster later, and the override should then apply. Expansion yielding nothing already makes it
inert, so keeping the row is correct rather than untidy.

**Tests:** removing a node deletes its node-scoped override in the same transaction; removing a node with
cascade also deletes the service-scoped overrides of its services; a cluster override survives every
removal; an override whose target was deleted out-of-band emits nothing.

### 4.9 Cross-store creation and reconciliation

Rule creation spans Grafana and Postgres, so the ordering and failure behaviour are part of the design:

1. **Mint `rule_id` first.** It is the idempotency key for every subsequent step; a retry is a plain upsert on the same id.
2. **Write Postgres, then Grafana.** If the Grafana call fails, the registry row is orphaned — which is already the reconciler's job, so no bespoke compensation path is needed.
3. **Grafana rule without a registry row** (the reverse failure, possible if a row is lost) is not silently harmful: the default is a literal in the rule, so it keeps evaluating correctly at defaults, and only the override API fails. Repair from the `pmm_rule_id` and `template_name` labels already stamped on the rule.

**The reconciler is a garbage collector, not a consistency mechanism — and it never writes to Grafana.**
Because Grafana is authoritative for which rules exist (§4.4) and both the API join and the collector's
ID→name resolution drop unresolvable rows at read time, it has **no correctness duty for alerting
behaviour**. It has two jobs, and only one of them costs anything real:

There are two kinds of orphan, and only one of them is the reconciler's problem:

| Orphan | Cleaned by | Cost of not cleaning it |
|---|---|---|
| Override rows whose **node/service** is gone | **The removal API hooks (§4.8)** — atomic, immediate | n/a — handled at the source |
| **Registry rows** whose Grafana **rule** is gone | **The reconciler.** Irreducible: rule deletion happens *in Grafana*, where PMM has no hook and gets no notification | **Real.** The collector keeps emitting `pmm_alert_threshold_override` series for a rule that no longer exists. Nothing queries them, but they consume VM ingestion and cardinality and grow without bound as rules churn |

So the reconciler has exactly **one** deletion job, plus a refresh:

It should also **refresh `grafana_rule_uid`** while it is there, since the listing already pairs each UID
with its `pmm_rule_id`.

An optional periodic sweep for stray override rows (direct SQL, or a removal path that forgets the hook)
is cheap — one `DELETE … WHERE scope='node' AND target NOT IN (SELECT node_id FROM nodes)` — but it is
belt-and-braces, not required, because read-time resolution already makes such rows inert. It must never
touch `scope='cluster'` (§4.8).

(The "an identical group re-POST is a no-op" measurement quoted in §7 belongs to the *rejected* inline
design, where a reconciler would re-render rules. It is not a licence for this one to write.)

> **A background job may not be needed at all.** The read path already fetches the authoritative rule
> list from Grafana, so it could delete registry rows it finds absent as a side effect — no ticker.
> Trade-off: cleanup then happens only when someone exercises the API, and it still needs the leader check
> (`IsLeader()`), because a follower must not write. Worth weighing rather than assuming the ticker.

Deletion safety rules:

- Require a **successful, complete** listing. On any error or partial response, skip the cycle — never delete on incomplete data.
- Require absence in **K consecutive cycles** (K ≥ 2) before deleting, to survive transient inconsistency and creation races.
- **Use a single global listing** (`GET /api/ruler/grafana/api/v1/rules` returns every folder) and match on the `pmm_rule_id` label alone. An earlier draft said to scope each check to a stored `folder_uid`; that is **wrong and would delete live data** — a user can move a rule between folders in Grafana, after which a per-folder check reports it absent and the reconciler removes the overrides of a rule that still exists. This is also the second reason `folder_uid` is not stored.

### 4.10 Multi-scope resolution — the shared resolver

Only node scope is implemented in the first increment, so the full resolver is a gate for the
**service/cluster increment**, not for the first merge. What must exist from day one is the **shared
function boundary**, so the API and the collector cannot drift:

```go
// ResolveEffective returns exactly one value per target, precedence applied.
// The collector and ListNodeThresholds MUST both call this. Nothing else may
// implement precedence.
func ResolveEffective(rule *models.AlertRule, param string,
    overrides []*models.AlertRuleThresholdOverride,
    inv Inventory) map[string]float64   // join-label value -> effective threshold
```

Invariants to assert and table-test:

- **Exactly one emitted series per `(rule, param, target)`.** This is not a tie-break preference: if the resolver ever emits two series with identical labels, the Prometheus gatherer errors and the **entire** `/metrics` response fails. `node_name` and `service_name` are both `UNIQUE` (`database.go:117`, `:139`), so live targets cannot collide — the risk is a resolver bug, so assert it.
- **Precedence `node` → `service` → `cluster`**, most specific first. A param legal at both service and cluster scope with both present resolves to **service**.
- **Cluster scope is only legal for params whose join label is service-level.** A node can host services from several clusters, so "the cluster override for this node" has no unique answer. This is a validation rule on `override_scopes`, not a runtime tie-break.
- **Unresolvable targets are skipped**, never defaulted to something else.

### 4.11 High availability

Every pmm-managed node registers collectors unconditionally and scrapes its own localhost, labelled
`instance = PMM_HA_NODE_ID`, so an N-node cluster emits N copies. Postgres is shared/external in HA, so
the copies are identical and `max by (<join>)` collapses them.

> **Invariant:** never drop `max by (<join label>)`. It looks redundant once the metric is override-only,
> but it is what makes the design HA-safe — and its absence only manifests with more than one node.

The **reconciler must be leader-only**:
`haService.AddLeaderService(ha.NewContextService("threshold-reconciler", fn))` (`haservice.go:569`).
`IsLeader()` returns true when HA is disabled, so single-node deployments are unaffected.

---

## 5. Worked examples

### 5.1 Multi-expression template — one line to make it overridable

`pmm_node_high_cpu_load` is already multi-expression on `main`. The whole template-side change:

```yaml
    params:
      - name: threshold
        summary: A percentage from configured maximum
        unit: "%"
        type: float
        range: [0, 100]
        value: 80
        overridable: true              # <-- the only addition
        override_scopes: [node]        # <-- optional; defaults to [node] / node_name
```

Generated rule, with one override (`node-02` → 90) and default 80:

| refId | body |
|---|---|
| `A` | `(1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100` *(unchanged)* |
| `T_threshold` | `max by (node_name) (label_replace(pmm_alert_threshold_override{rule_id="7f3a…", param="threshold"}, "node_name", "$1", "target", "(.*)")) or (max by (node_name) (last_over_time(pmm_managed_inventory_nodes[15m])) * 0 + 80)` |
| `C` | `$A > $T_threshold` |

Resolves to `node-02 = 90`, every other node `= 80`.

> **The `T_<param>` step is always emitted, including when no override rows exist.** An earlier draft
> proposed omitting it so that an untuned rule stayed byte-identical to today's output. That is
> **wrong and was removed**: a rule created without the step would need its definition changed to add
> one when the first override arrives — which resets alert state for every instance on that rule, the
> precise failure this design exists to avoid. With no overrides the step is still present and simply
> resolves to the default for every target.
>
> The honest cost: **every overridable rule pays the fan-out scan (~N × 91 samples per evaluation)
> whether or not anyone has tuned it.** The "untuned rules cost nothing" property does not exist.

### 5.2 Single-expression template — desugared

`pmm_mysql_too_many_connections` on `main` is single-expression:

```yaml
    expr: |-
      max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
      mysql_global_variables_max_connections
      * 100
      > bool [[ .threshold ]]
```

Marking its param `overridable: true` triggers desugaring at build time — the template file keeps its
`expr:` form; only the generated rule changes:

| refId | body |
|---|---|
| `A` | `max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job) mysql_global_variables_max_connections * 100` |
| `T_threshold` | the threshold expression (join label `service_name`, fan-out over `pmm_managed_inventory_agents{service_name!=""}`) |
| `C` | `$A > $T_threshold` |

Two mechanical details:

1. **`> bool` disappears** — Grafana math comparisons already yield 0/1.
2. **`{{ $value }}` must become `{{ printf "%.2f" $values.A.Value }}`** — with multiple refIDs `$value` is no longer a single scalar. This template's `description` uses `{{ $value }}`, so shipping desugaring without the rewrite ships broken alert text. Rewrite it during desugaring (annotations are filled at `service.go:744`).

Constraints inherited from parsing an `expr:`: the token must be the RHS of the **final** comparison, and
a single expression admits **one** overridable param. Reject at parse time otherwise.

#### Do the split on the AST, not with a regular expression

A permissive regex split is the wrong tool: it mishandles parentheses, comparison modifiers (`bool`),
vector-matching clauses (`on`/`ignoring`, `group_left`), and nested comparisons, and it fails *silently*
by producing a plausible-but-wrong `A`. `github.com/prometheus/prometheus v0.313.0` is **already a direct
dependency on `main`** (`go.mod:61`), so the parser is available with no new dependency.

The one wrinkle is that `[[ .threshold ]]` is not valid PromQL, so it must be replaced before parsing.
Pad the replacement to the **same byte length** as the token and AST positions map 1:1 onto the original
text — which means `A` can be sliced out of the *original* string, preserving the author's formatting
exactly:

```go
// splitSingleExpr splits a single-expression template into the observed query and
// the comparison operator, so the builder can emit A + T_<param> + math C.
//
// The [[ .param ]] token is replaced by a byte-length-padded numeric sentinel so
// that PromQL positions map 1:1 onto the original text.
func splitSingleExpr(expr, paramName string) (lhs string, op parser.ItemType, isBool bool, err error) {
    token := paramTokenRegexp(paramName).FindString(expr)
    if token == "" {
        return "", 0, false, fmt.Errorf("param %q is not referenced in the expression", paramName)
    }

    // "0" plus spaces to the token's exact byte length: parseable, and position-preserving.
    sentinel := "0" + strings.Repeat(" ", len(token)-1)
    probe := strings.Replace(expr, token, sentinel, 1)

    parsed, err := parser.ParseExpr(probe)
    if err != nil {
        return "", 0, false, fmt.Errorf("failed to parse expression: %w", err)
    }

    // The threshold must be the RHS of the outermost comparison.
    bin, ok := parsed.(*parser.BinaryExpr)
    if !ok || !bin.Op.IsComparisonOperator() {
        return "", 0, false, errors.New("an overridable param must be the right-hand side of the expression's top-level comparison")
    }
    if num, ok := bin.RHS.(*parser.NumberLiteral); !ok || num.Val != 0 {
        return "", 0, false, errors.New("an overridable param must be compared directly, not used inside a larger expression")
    }

    // Vector matching on the comparison itself cannot survive the split into a
    // Grafana math step, so reject it rather than silently changing semantics.
    if bin.VectorMatching != nil {
        return "", 0, false, errors.New("vector matching on the threshold comparison is not supported for overridable params")
    }

    // Positions are byte offsets into probe, which is the same length as expr.
    r := bin.LHS.PositionRange()
    return strings.TrimSpace(expr[r.Start:r.End]), bin.Op, bin.ReturnBool, nil
}
```

Everything the regex approach would have to special-case is now either handled by the parser or rejected
with a precise message. Note `bin.ReturnBool` captures the `bool` modifier — which is then **dropped**,
because Grafana math comparisons already yield 0/1 (§5.2, gotcha 1).

**Verification must precede implementation.** The filter-vs-0/1 semantic difference is not proven for
real alert lifecycle behaviour — `for:` pending/firing transitions, NoData, recovery, annotation
rendering. Desugaring an existing template before that check risks changing *alerting behaviour* while
believing you only changed *threshold delivery*. See §9, gate 3.

### 5.3 Two params at two different scopes — validated live

The case that proves the design generalises. Both params in one rule, with different join labels:

```yaml
    queries:
      - ref_id: A                       # service-scoped (carries node_name too)
        expr: |-
          sum by (service_name, node_name) (pg_stat_database_numbackends)
          / on (service_name, node_name) group_left()
          max by (service_name, node_name) (pg_settings_max_connections) * 100
      - ref_id: B                       # node-scoped
        expr: |-
          (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
    expressions:
      - ref_id: C
        type: math
        expression: "$A > [[ .connections_threshold ]] && $B > [[ .cpu_threshold ]]"
    condition: C
    params:
      - name: connections_threshold
        value: 80
        overridable: true
        override_scopes: [service, cluster]   # -> join label service_name
      - name: cpu_threshold
        value: 80
        overridable: true
        override_scopes: [node]               # -> join label node_name
```

Deployed as four rules on a live server (actuals: connections ≈ 0.7 %, CPU ≈ 5.4 %):

| Case | Overrides | Effective thresholds | Result |
|---|---|---|---|
| none | — | conn **0.50** / cpu **3.00** (both fan-out defaults) | **fires** |
| node only | `cpu_threshold`@node = 90 | conn 0.50 / cpu **90** | does not fire |
| service only | `connections_threshold`@service = 90 | conn **90** / cpu 3.00 | does not fire |
| node **+** service | cpu@node = 1, conn@service = 0.1 | conn **0.10** / cpu **1.00** | **fires** |

The last row is the load-bearing one: its **defaults are 90/90**, so it could never fire on defaults. It
fired, annotated `conn=0.70% (thr 0.10) cpu=5.36% (thr 1.00)` — two overrides at two different scopes
applied simultaneously in one rule.

This also proved **Grafana math subset-matches across differing label sets**: `$A{node,service}` against
`$T{service}`, `$B{node}` against `$T{node}`, then `&&` across the two results. That had been an
assumption in every prior write-up.

### 5.4 Annotations

`{{ printf "%.0f" $values.T_<param>.Value }}` renders the **effective per-node** threshold — verified,
including on a multi-param rule. Templates should use it rather than interpolating `[[ .threshold ]]`,
which freezes the default at creation time and is then wrong for every overridden target. Word it
neutrally (`"CPU load is X%, threshold Y%"`), because annotations are also rendered for tracked
non-firing instances.

---

## 6. Implementation plan

Ordered so each step is independently reviewable. All paths are `main` paths.

| # | Step | Files | Effort |
|---|---|---|---|
| 1 | `overridable` + `override_scopes` on `Parameter`; validation (float only, must be a `[[ .name ]]` token in an expression step, at most one for a desugared single-expression template) | `managed/pi/alert/parameter.go:25`, `managed/pi/alert/query.go` validation, `template.go:113` | S |
| 2 | Migration 119 + reform models + helpers (`FindThresholdOverrides`, `FindThresholdOverridesByTarget`, `UpsertThresholdOverride`, `DeleteThresholdOverride`, `FindAlertRules`, `CreateAlertRule`, `DeleteAlertRule`) | `managed/models/database.go:1184`, new `alert_rule*_model.go` + `alert_rule_helpers.go`, `make gen` | M |
| 3 | Threshold collector: one indexed query + bounded ID→name resolution; fixed 3-label desc; direct `Describe`; register on the default registry | new `managed/services/alerting/threshold_metrics.go`, `main.go` near `:942` | S |
| 4 | Rule builder: allocate `T_<param>` refIDs, inject the threshold step, swap `[[ .param ]]` → `$T_<param>`; thread `ruleID` through | `rule_builder.go:58`/`:81`, reusing `newPromQueryData:141` and `newMathExpressionData:165` | M |
| 5 | `CreateRule`: mint `rule_id`, stamp a `pmm_rule_id` label, persist the registry row with the `params` snapshot, return the id. Also have `CreateAlertRule` parse and return the created Grafana UID (already in the POST response body) and store it as `grafana_rule_uid` | `service.go:679`, `grafana/client.go:712` | S |
| 6 | Single-expression desugaring: split `expr` into `A` + `T_<param>` + math `C`; rewrite `{{ $value }}` → `{{ $values.A.Value }}` | new `managed/pi/alert/overridable.go`, `rule_builder.go`, annotations at `service.go:744` | S–M |
| 7 | API: `overridable` on the param-definition message; `rule_id` on `CreateRuleResponse`; list/set/delete threshold RPCs with `scope`/`target` fields present but node-only behaviour; shared precedence function. **Route shape is pending §10 Q1** — node-centric vs generic — and that answer is needed before the proto freezes | `api/alerting/v1/alerting.proto`, new `managed/services/alerting/threshold_overrides.go`, `make gen` | M |
| 8 | `ListAlertRules` on the Grafana client + leader-only reconciler (registry-row orphans only). Separately, hook `DeleteThresholdOverridesForTarget` into `RemoveNode` / `RemoveService` (§4.8) | `managed/services/grafana/client.go:712` area, `deps.go`, `haservice.go:569`, `models/node_helpers.go:255`, `models/service_helpers.go:354` | S |
| 9 | Mark built-in templates overridable — `node_high_cpu_load.yml` (one line) and, once step 6 lands, `mysql_too_many_connections.yml` | `managed/data/alerting-templates/` | XS |
| 10 | Add `cluster` to the inventory services descriptor, before anything depends on that label set | `managed/services/inventory/inventory_metrics.go:84` | XS |

**3–5 weeks** for one engineer for the node-only increment.

The earlier figure of 2–3 weeks covered the ten steps above with unit tests and silently excluded
everything else. Stated properly:

| Included | Excluded |
|---|---|
| The ten steps, with unit tests | UI |
| `make gen` cycles for proto and reform | Service and cluster **behaviour** (schema/API only) |
| AST-based desugaring (§5.2) and its rejection cases | Scale re-measurement on a large inventory |
| Cross-store failure handling and the reconciler (§4.9) | HA cluster testing beyond the `max by` invariant |
| GC for stale targets (§4.8) | API review turnaround |

Assumes the §9 decision gates are resolved **in parallel** with implementation, not serially. If gate 1
(NoData/availability) or gate 5 (precedence sign-off) blocks, the critical path extends by that wait
rather than by engineering time.

Deliberately **not** needed: `UpdateAlertRule`, a per-group write mutex, `uid`-preserving JSON patching,
Grafana provenance, PromQL parsing of generated text, a VM write path, tombstones, or a keep-alive writer.

### Follow-ups, costed

| Follow-up | Effort | Why not now |
|---|---|---|
| Dedicated endpoint + own scrape job, giving `up{job="pmm-thresholds"}` | XS–S | Isolation and hygiene, not behaviour. `promscrape.yml` is generated in pmm-managed (`victoriametrics.go` `populateConfig`), so it stays in-tree. |
| Service and cluster **behaviour** | M | Schema and API already generalised; needs the precedence function plus the `cluster` inventory label from step 10. |
| UI | M | Separate deliverable. |

---

## 7. Alternatives rejected

Each with the measurement that settled it.

| Alternative | Why not |
|---|---|
| **Inline the overrides as literals in the rule query** (no metric) | A rule-definition change **resets alert state for every instance on that rule** — `activeAt` moved `14:09Z → 14:18Z` for a node whose own threshold was untouched, with `uid` *and* `guid` preserved. Under `for: 5m`, tuning one node blinds every other node on that rule for five minutes. Batching, one-rule-per-group, shortening `for:` and `keep_firing_for` all fail to preserve the timer, because Grafana keys state on the rule definition. |
| **Put the canonical state inside the rule** (no tables) | Same state-reset cost, plus no contractual home: either parseable PromQL (symmetric escaping of user-controlled `node_name`, stable float formatting, permanent support for old shapes) or a JSON sidecar in the query model. The sidecar survives both an API round-trip and a UI edit on Grafana 12.4.5 — but rests on an undocumented implementation detail that an upgrade could remove silently. |
| **Read-only rules via Grafana provenance** | Group-wide and irreversible: claiming provenance makes the Ruler API reject the **whole group** (`400`), and release fails with `409 alerting.provenanceMismatch`. Only exit is delete-and-recreate, minting a new UID and breaking silences. |
| **Emit a threshold for every node** (default or override) | The naive shape: `rules × params × nodes` ≈ 20,000 series and ~2,000 samples/s at 1,000 nodes × 20 rules, ~99 % of it identical defaults, plus `2 + R` queries inside every 10 s scrape sharing a 9 s timeout with the inventory and HA collectors. |
| **Push overrides into VM on change** (import API / remote-write) | Buys only propagation latency — immediate vs ≤ 1 scrape — which is noise against a 60 s evaluation interval. Costs a keep-alive writer, an explicit **tombstone** on clear, and retry/ordering logic. |
| **Cache overrides in a `GaugeVec` mutated on write** | Needs restart warm-up and produces N duplicated series in HA, for no gain: after override-only emission the read is one indexed query over a tens-of-rows table. |
| **A Prometheus-compatible "PMM Thresholds" datasource** | Architecturally cleanest — zero duplication, immediate clears, precedence purely in Go — but the largest new surface (a hand-written API shim whose contract with Grafana's Prometheus datasource is version-sensitive), it touches `build/ansible`, and it puts a synchronous Postgres round-trip inside every rule evaluation. Revisit if thresholds must change without an API call (schedules, computed baselines). |
| **Exception-rule splitting** / thresholds as scrape-target labels | Rule splitting rewrites the base rule anyway and changes alert identity when a target moves between rules, breaking dedup and silences. The label approach needs a vmagent reload per override change and pollutes every scrape target. |

### Known costs of the recommendation

- **Two stores.** A rule deleted in Grafana orphans an `alert_rules` row; paid for by the reconciler (step 8), which is safe to run on a timer because an identical group re-POST is a genuine no-op (`"no changes detected in the rule group"`, `activeAt` preserved).
- **Clear latency ~5 min**, governed by VM's lookbehind. Bounding it would couple the generated query to the user-configurable `MR` setting, so accept it and say so in the UI.
- **Query scan is ~1.5× the naive design**: the fan-out reads N series per rule per evaluation (`V × 61 + N × 91 ≈ 93k` samples vs `N × 61 ≈ 61k`). Traded for ~600× less ingestion and ~600× fewer stored series — the right trade, since ingestion is persistent and scan is transient, but not free.
- **The default is a literal in the rule**, so changing a default for everyone is a rule edit.
- **Threshold history lives only as long as VM retention** (~30 days).

---

## 8. Verification

Already verified on a live server (Grafana 12.4.5, VM v1.147.0, 3 nodes / 1 service):

| Verified | Result |
|---|---|
| Threshold expression: no overrides → all targets at default; one override → that target only | ✅ |
| Override that **raises** the bar suppresses a target the default would have fired | ✅ |
| Target in inventory but reporting no data → no alert instance | ✅ harmless |
| Node + service overrides in one rule, two join labels | ✅ §5.3 |
| Grafana math subset-matches across differing label sets | ✅ |
| Clearing an override needs no tombstone | ✅ measured at 280–300 s |
| Override series expire independently per `(rule, param, target)` | ✅ |
| `$values.T_<param>.Value` in annotations | ✅ multi-param |
| `max by` adds no scan cost; `last_over_time` scans the full range | ✅ 61 / 91 / 361 / 11,246 samples per series |

Still to verify:

1. **Filter vs 0/1 semantics after desugaring** — a PromQL comparison without `bool` filters series, Grafana math yields 0/1. The firing set should match; confirm against a real rule including `for:` and NoData. Live check, not a unit test.
2. **Behaviour at scale** — all measurements were on 3 nodes and 1 service; the cardinality and scan figures are extrapolations. The fan-out reading N series per evaluation is the number most worth re-measuring.
3. **Three scopes on a single param**, resolved most-specific-first in Go.
4. **`-promscrape.config.dryRun` acceptance** of a new scrape job, if the dedicated-endpoint follow-up is taken.

Procedure: [`dev/docs/process/running-and-verifying-locally.md`](dev/docs/process/running-and-verifying-locally.md).

---

## 9. Decision gates

Promoted out of "open questions" because each can invalidate GA behaviour. None is an engineering
unknown; each needs a decision or a test result.

| # | Gate | Owner | Blocks |
|---|---|---|---|
| 1 | **`no_data_state` / availability.** Rules resolve NoData → **OK**, so a missing threshold reports `health=ok` while silently not evaluating (measured). `last_over_time` bounds the exposure to the window; `up{job="pmm-managed"}` detects it. Flipping to `NoData` is not viable — it would fire for every legitimately absent target. Decide whether window + existing health signal is sufficient for GA. | Product + Eng | GA |
| 2 | **Stale-target GC** implemented and tested per §4.8 | Eng | merge |
| 3 | **Desugaring semantics verified live** — firing, pending under `for:`, recovery, NoData, annotation rendering — *before* step 6 is implemented | Eng | step 6 |
| 4 | **Cross-store creation semantics** implemented per §4.9 (ordering, idempotency key, K-cycle absence rule) | Eng | merge |
| 4b | **Registry-row orphan GC** — rows for rules deleted in Grafana, so emitted series do not grow without bound. Narrowed: override rows for deleted nodes/services are handled by the removal API hooks (§4.8), not here | Eng | GA |
| 5 | **Multi-scope precedence signed off** (`node` → `service` → `cluster`; cluster legal only for service-level join labels) *before* the proto freezes | Product | API freeze |
| 5b | **API style chosen** — node-centric vs generic routes (§10 Q1). Routes are additive-only, so deciding late means carrying both families forever | Product + Eng | API freeze |
| 6 | **Shared resolver** (§4.10) is the single implementation of precedence, table-tested across scope combinations | Eng | service/cluster increment |
| 7 | **Fan-out re-measured at representative scale and under HA** | Eng | GA |
| 8 | **Every overridable param emits its `T_<param>` step at creation** — no conditional omission | Eng | merge |

---

## 10. Open questions

Everything that gates GA or the API freeze has moved to §9, **except the first item below, which is
open but time-critical**: it must be settled before the proto is frozen.

### 1. Which API style — node-centric routes, or generic scope/target routes?

**Undecided, and urgent.** HTTP routes are additive-only, exactly like proto fields: whichever style
ships first can never be removed. So if node-centric routes ship now and generic routes are added when
service and cluster scope land, PMM carries **both families indefinitely**. Choosing later is not
neutral — it is choosing duplication.

Note this is a *different* question from the one already settled. The decision to "generalise the schema
and API now, with node-only behaviour" covers the **message fields** (`scope`/`target` alongside
`node_id`). It does not cover the **paths**, and the paths are the half that cannot be changed afterwards.

**Style A — node-centric** (what the feature branch implements):

```
GET    /v1/alerting/nodes/{node_id}/thresholds
POST   /v1/alerting/nodes/{node_id}/thresholds
DELETE /v1/alerting/nodes/{node_id}/thresholds/{rule_id}/{param_name}
```

**Style B — generic:**

```
GET    /v1/alerting/thresholds?scope=node&target=<id>[&rule_id=<id>]   # all filters optional
POST   /v1/alerting/thresholds                    body: {scope, target, rule_id, param_name, value}
DELETE /v1/alerting/thresholds?scope=&target=&rule_id=&param_name=
POST   /v1/alerting/thresholds:batchUpdate        body: {updates: [...]}      # optional, see below
```

| | Style A — node-centric | Style B — generic |
|---|---|---|
| Adding service/cluster scope later | needs a second route family, kept forever | no new routes |
| Discoverability / REST shape | ✅ clearer resource hierarchy | ⚠️ filters are less self-describing |
| Serves admin and debug reads ("all overrides for rule R", "everything") | ✅ needs extra endpoints | ✅ one route, optional filters |
| UI must send `scope` explicitly | no | yes (trivial) |
| Matches the branch, so no UI rework | ✅ | ⚠️ small client change |

**Three facts that constrain the choice regardless of which style wins:**

1. **`target` cannot be a path segment.** `node_id` and `service_id` are opaque IDs, but a `cluster` target is an **arbitrary label value** — `prod/us-east` breaks grpc-gateway path matching outright (path params do not match `/` without `{target=**}`), and spaces or unicode need encoding. PMM's own convention agrees: every existing path segment in `api/` is an opaque ID, never a free-form value. So `/thresholds/{scope}/{target}` is not an option in either style.
2. **A batch method is idiomatic here.** PMM already uses AIP-style custom verbs — `/v1/accesscontrol/roles:assign`, `/v1/actions:startServiceAction`, `/v1/advisors/checks:batchChange`, `/v1/inventory/services:getTypes`. This matters because the UI diffs a whole modal of rows on submit and currently fires **N separate set/delete calls**, so a partial failure leaves the modal half-applied with no way to report which rows landed. `checks:batchChange` is the in-repo precedent for making that one transactional call. Worth deciding alongside the style, since it is the same proto change.
3. **`rule_id` is not a unique key in the response.** Duplicated rules in Grafana share a `pmm_rule_id` (§4.4), so a list can legitimately return two entries with the same `rule_id` *and* `param_name`, differing only in rule title. Nothing breaks — the field is `repeated` — but a client keying a map on `(rule_id, param_name)` would silently collapse them. State it in the proto comments rather than leaving it to be discovered.

**Recommendation if a tie-break is needed:** Style B, on the strength of the additive-only argument
alone — the cost of being wrong is permanent route duplication, versus a one-off loss of REST
readability. But this is an API-review call, and it belongs with gate 5's proto freeze in §9.

### Genuinely open, not blocking

2. **Audit history.** Nothing records "the threshold when this fired" beyond VM retention (~30 days). If
   that is required, a small append-only table beats shaping the primary design around it.
3. **Do any planned overridable templates resist the `$X <op> [[ .param ]]` shape?** The AST split
   (§5.2) rejects a threshold used inside arithmetic, in a subquery, in a nested comparison, or with
   vector matching on the comparison itself. Worth sketching one candidate before step 6 lands, so the
   rejection set is validated against real intent rather than assumed.
4. **Whether an immediate delete hook should accompany the reconciler GC** (§4.8) for faster UI feedback.
   It cannot replace the reconciler pass — removals can happen while pmm-managed is down, and in HA only
   the leader may write — so this is a UX question, not a correctness one.
5. **Dynamic defaults.** The default is a literal in the rule, so changing it for everyone is a rule
   edit. If defaults ever need to change without one — schedules, computed baselines — that is the
   trigger to revisit the thresholds-datasource alternative in §7, where the whole computation is in Go.

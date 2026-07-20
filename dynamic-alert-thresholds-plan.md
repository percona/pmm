# Dynamic Alert Thresholds (per-node overrides)

## Context

PMM alert rules created from templates bake the threshold literal into the query
(`... > 80`). Changing a threshold for specific nodes (e.g. a node whose CPU
runs hot due to HW constraints) means editing/recreating the rule. We want a
**UI/API way to override a rule's threshold per node without touching the rule
query**.

The approach: make the threshold a **custom metric** that the alert compares
against. PMM emits `pmm_alert_threshold{rule_id, param, node_name}` (override
value where set, template default otherwise); the rule evaluates
`observed > pmm_alert_threshold`. Updating a node's threshold becomes a data
change (one row), never a rule-query change.

**This plan targets branch `PMM-14911-alerts-be`**, which already implements the
Grafana multi-expression (queries `A`/`B`, math expression `C`, `condition`)
foundation. That branch proved the `$A > $B` evaluation path with a *hardcoded*
`B` (`label_replace(vector(10), "node_name", "pmm-server", …)`) that was then
removed (commit `0ebc2b133`). This plan replaces that stub with a **real,
DB-backed, per-node custom metric** plus the API to edit it.

### Scope decisions (confirmed with user)
1. **Node-level only** — overrides keyed by the stable inventory **`node_id`**;
   `node_name` (fixed `join_label` in v1) is resolved from inventory for the
   emitted metric label.
2. **Registry + reconcile** — persist a PMM-side rule record; a background
   reconciler drops records whose Grafana rule no longer exists (no
   `DeleteRule` RPC exists today).
3. **Auto-inject B via token-swap** — templates stay clean (`$A > [[ .threshold ]]`
   + `overridable: true`); PMM injects the `pmm_alert_threshold` query and swaps
   the token. No PromQL parsing.
4. **Backend + API** — DB, collector, rule-builder wiring, gRPC override CRUD.
   No frontend (separate follow-up).
5. **Node-centric API** — the API is keyed on the node, not the rule: you list a
   node's thresholds (defaults + overrides) and set/clear an override for that
   node. A single threshold's identity within a node is still `(rule_id,
   param_name)` because a threshold is defined by a rule's overridable param.

## What already exists on the branch (reuse, do not rebuild)
- Multi-expr template model: `managed/pi/alert/query.go` (`TemplateQuery`,
  `TemplateExpression`, `UsesMultipleExpressions`, `validateMultipleExpressionSteps`),
  `Template` struct in `managed/pi/alert/template.go`.
- Rule builder: `managed/services/alerting/rule_builder.go`
  (`buildGrafanaRuleData` / `buildMultiExpressionRuleData`, `newPromQueryData`,
  `newMathExpressionData`, `fillAndFilterExpr`). Filter fix already preserves
  label-less/threshold series (`label_match(%s,%q,"(%s)|")`).
- Rule DTO: `managed/services/alert_rule.go` (`Data.Model` is `json.RawMessage`).
- `CreateRule`: `managed/services/alerting/service.go:679` → calls
  `buildGrafanaRuleData` at ~732, `grafanaClient.CreateAlertRule` at ~794,
  returns empty `CreateRuleResponse` at ~799.
- Collector pattern: `managed/services/ha/ha_metrics.go`,
  `managed/services/inventory/inventory_metrics.go` (DB-backed collector via
  `InTransactionContext` + `models.FindNodes`). Registered in
  `managed/cmd/pmm-managed/main.go` (~933, `prom.MustRegister`), served on
  `/debug/metrics` (127.0.0.1:7773) — **already scraped by VictoriaMetrics**
  (`managed/services/victoriametrics/prometheus.go:80`).
- Node inventory: `models.FindNodes(q, models.NodeFilters{})`
  (`managed/models/node_helpers.go`), `Node.NodeID` (PK) / `Node.NodeName`
  (`managed/models/node_model.go`).
- Migration + reform conventions: `managed/models/database.go`
  (`databaseSchema [][]string`), reform model example
  `managed/models/action_models.go` (BeforeInsert/BeforeUpdate/AfterFind,
  `models.Now()`).

## Implementation

### 1. Template param metadata — mark overridable params
- `managed/pi/alert/parameter.go`: add `Overridable bool `yaml:"overridable,omitempty"``
  to `Parameter`.
- Validation (in `template.go` `validateSteps`/`validateParams`, or a new
  `validateOverridable`): a param with `overridable: true` must
  (a) be `Float`, (b) belong to a **multi-expression** template, and
  (c) appear as a `[[ .name ]]` token inside an `expressions[].expression`
  string. Reject otherwise. Reuse `UsesMultipleExpressions()`.
- Persist + expose: add `Overridable` to `AlertExprParamDefinition`
  (`managed/models/template_model.go`) and carry it in `ConvertParamsDefinitions`
  (`managed/models/template_helpers.go`). Add an `overridable` field to the
  param-definition message in `api/alerting/v1/alerting.proto` so the UI can
  tell which params are override-eligible; regenerate with `make gen`.
- Mark built-in templates (start with `node_high_cpu_load.yml`) `overridable: true`
  on the threshold param; keep them in clean `$A > [[ .threshold ]]` form.

### 2. Rule builder — inject the custom-metric query + swap token
In `managed/services/alerting/rule_builder.go`, thread `ruleID string` and the
overridable param set into `buildMultiExpressionRuleData`. For each overridable
param `P`:
- Allocate a collision-free refID (reserved prefix, e.g. `T_<P>`; check against
  existing query/expression refIDs).
- Append a prom query node via `newPromQueryData`:
  `pmm_alert_threshold{rule_id="<ruleID>", param="<P>"}` (rule_id/param are
  literals, not template tokens).
- Before `fillExprWithParams` runs on each `expressions[].expression`, replace
  the literal token `[[ .P ]]` with `$T_<P>`. The default value is therefore
  **not** placed in the expression — it is emitted by the collector instead.
  (Unreferenced `P` in the params map is fine; `missingkey=error` only fires on
  referenced-missing keys.)
- Non-overridable params keep flowing through `fillExprWithParams` unchanged.
Result e.g. for CPU+memory: two `pmm_alert_threshold{param=…}` queries + math
`"$A > $T_cpu_threshold && $B > $T_memory_threshold"`.

### 3. Rule identity + registry (persist at CreateRule)
`managed/services/alerting/service.go` `CreateRule`:
- Mint `ruleID := uuid.New().String()` (google/uuid already used in repo).
- Stamp `labels["pmm_rule_id"] = ruleID` alongside existing
  `percona_alerting`/`template_name` labels.
- Pass `ruleID` into `buildGrafanaRuleData`.
- After successful `CreateAlertRule`, insert an `alert_rules` row
  (rule_id, template_name, folder_uid, rule_group, rule_title,
  `default_params` = JSON map of overridable param → default value, timestamps).
  Storing defaults at creation makes the collector/API independent of later
  template edits.
- Add `rule_id` to `CreateRuleResponse` (proto) so clients learn the id.

### 4. DB — two tables + reform models
- Migration: new `databaseSchema` version in `managed/models/database.go`:
  - `alert_rules(rule_id PK, template_name, folder_uid, rule_group, rule_title,
    default_params JSONB, created_at, updated_at)`
  - `alert_rule_threshold_overrides(id PK, rule_id FK→alert_rules ON DELETE
    CASCADE, param_name, node_id FK→nodes ON DELETE CASCADE, value DOUBLE
    PRECISION, created_at, updated_at, UNIQUE(rule_id, param_name, node_id))`
    — keyed by stable `node_id`; removing a node auto-clears its overrides.
- Reform models (new files `managed/models/alert_rule_model.go`,
  `managed/models/alert_rule_threshold_override_model.go`) mirroring
  `action_models.go` (reform tags, `BeforeInsert`/`BeforeUpdate`/`AfterFind`,
  `models.Now()`). Add helpers following `node_helpers.go`:
  `FindAlertRules`, `CreateAlertRule`, `FindOverridesByNode(nodeID)`,
  `FindOverridesByRule(ruleID)`, `UpsertOverride`, `DeleteOverride`,
  `DeleteAlertRule`.

### 5. Collector — emit `pmm_alert_threshold`
New `managed/services/alerting/threshold_metrics.go`:
- `AlertThresholdCollector` implementing `prom.Collector`. Fixed label set in v1
  → `[rule_id, param, node_name]` via `prom.NewDesc`; `Describe` via
  `prom.DescribeByCollect` (keeps it easy to make unchecked later if service-level
  join labels are added).
- `Collect`: in one `InTransactionContext`, load all `alert_rules`; enumerate
  nodes once via `models.FindNodes(tx.Querier, NodeFilters{})` and build a
  `node_id → NodeName` map; for each rule × each overridable param (from
  `default_params`) × each node, emit
  `MustNewConstMetric(desc, Gauge, override?override:default, ruleID, param, node.NodeName)`.
  Overrides are stored by `node_id`; the emitted label is the resolved
  `node_name` (the join label the rule query uses).
- Register in `managed/cmd/pmm-managed/main.go` (~933) with `prom.MustRegister`.
- Emitting the default for **every** node is deliberate: Grafana math `$A > $B`
  inner-matches by label, so B must have a value for every node A can produce,
  else unoverridden nodes silently never alert.

### 6. Reconcile — drop orphaned registry rows
- Add `ListAlertRules` (GET `/api/ruler/grafana/api/v1/rules/{folder}`) to the
  grafana client (`managed/services/grafana/client.go`) + `deps.go` interface —
  only `CreateAlertRule` exists today.
- Periodic reconciler (ticker goroutine in the alerting `Service`): list Grafana
  rules, collect live `pmm_rule_id` labels, delete `alert_rules` rows (cascade
  removes overrides) whose id is absent. Handles Grafana-side deletes without a
  PMM `DeleteRule` RPC.

### 7. API — node-centric threshold overrides
The API entry point is the **node**, not the rule (thresholds are tuned per
node). A single threshold is still identified by `(rule_id, param_name)` inside
node-scoped calls, since a threshold is defined by a rule's overridable param.
`api/alerting/v1/alerting.proto` + service methods in
`managed/services/alerting/`:
- **`ListNodeThresholds(node_id)`** → for that node, one entry per overridable
  threshold across active rules, each carrying: `rule_id`, `rule_title`,
  `template_name`, `param_name`, `summary`, `unit`, `default_value`,
  `effective_value`, `is_overridden`. **Returns defaults as well as overrides**
  in a single call so the UI shows the full picture. (Data source: `alert_rules`
  registry for rules + their `default_params`, left-joined to
  `alert_rule_threshold_overrides` filtered by `node_id`.)
- **`SetNodeThreshold(node_id, rule_id, param_name, value)`** → upsert override.
  Validate: node exists, rule exists and the param is `overridable` on it, value
  within the param's range.
- **`DeleteNodeThreshold(node_id, rule_id, param_name)`** → remove the override →
  the node reverts to the template default automatically.
- (Optional) `ListRules` to enumerate PMM-created rules for admin/debug views.
- `make gen` after proto changes; wire into the gRPC server registration used by
  the alerting service.
- **v1 simplification**: every overridable `(rule, param)` is treated as
  applicable to every node (the collector already emits per node for all nodes),
  so `ListNodeThresholds` returns all overridable thresholds. Scoping the list to
  only rules whose filters actually target the node is a follow-up (requires
  evaluating rule filters against the node).

## Verification
- Unit: extend `managed/services/alerting/rule_builder_test.go` for the
  token-swap + injected query (single and multi-threshold); add param-validation
  cases in `managed/pi/alert` for `overridable` (reject non-float / single-expr /
  token-absent); collector emit test (override-by-node_id vs default, correct
  node_name label); API validation tests (unknown node/rule/param, range).
- `make gen && make format`; `go test ./managed/pi/alert/...
  ./managed/services/alerting/... ./managed/models/...`.
- End-to-end (managed DB test harness — run a golang container with
  `--network container:pmm-server --volumes-from pmm-server`):
  1. Create a rule from an `overridable` template via `CreateRule`; capture the
     returned `rule_id`.
  2. `ListNodeThresholds(<pmm-server node_id>)` → shows the threshold with
     `default_value=80`, `effective_value=80`, `is_overridden=false`.
  3. `curl 127.0.0.1:7773/debug/metrics | grep pmm_alert_threshold` → one series
     per node at the default (80).
  4. `SetNodeThreshold(<pmm-server node_id>, rule_id, "threshold", 50)`;
     re-list → `effective_value=50`, `is_overridden=true`; re-curl → the
     pmm-server series is 50, others still 80.
  5. Query VictoriaMetrics for `pmm_alert_threshold` to confirm the scrape landed.
  6. Inspect the created Grafana rule: query `B` = `pmm_alert_threshold{rule_id=…}`,
     expression `C` = `$A > $B`; confirm it fires for pmm-server at >50 but others
     only at >80.
  7. `DeleteNodeThreshold(...)` → reverts to 80. Delete the rule in Grafana →
     reconciler removes the `alert_rules` row and the series disappear next scrape.

## Follow-ups (out of scope)
- Frontend for editing per-node thresholds.
- Scope `ListNodeThresholds` to rules that actually target the node (filter eval).
- Service-level (`service_name`) and arbitrary-label overrides (would make the
  collector's label set variable → switch to an unchecked collector).
- `UpdateRule`/`DeleteRule` RPCs for full PMM-owned lifecycle (reconcile covers
  cleanup for now).

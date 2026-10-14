# PMM-15663 design: scoped annotations on All

Status: draft for review, 14 Oct 2026. Scratch branch `PMM-15663-annotation-bench`, not for merge.

## 1. Problem and scope

[percona/pmm#6045](https://github.com/percona/pmm/pull/6045) (PMM-15609) sets `allValue: ".+"` on fleet-sized
`node_name` and `service_name` variables. Dashboard annotations match Grafana tags exactly, so on All the tag is the
literal `.+` and node- and service-scoped annotations disappear.

This design fixes PMM-15663 only. It must not block two linked tickets, which stay out of scope:

- PMM-15073: edit or delete a scoped annotation from one place, a scope selector in the UI.
- PMM-15144: automatic alert annotations scoped to a service or node, firing to resolve as a region.

## 2. Requirements

From PMM-15663:

1. With variables on All, node- and service-scoped annotations appear on the matching dashboards, and no request URL
   exceeds 10 KB at 5,000 services.
2. With a node or service selected, only its own and global annotations appear.
3. Existing annotations keep working after upgrade.

Agreed in the earlier sessions and this one:

- LBAC is pass/fail: a restricted user sees exactly the scoped annotations of what the user may see.
- Scale: 500 scoped annotations over 30 days normally, 5,000 as stress; 200k series normally, 3M as stress; 1,000 nodes,
  8,000 services.
- Scoped annotations follow metric retention (30 days by default).
- Global annotations (`pmm_annotation` tag) stay in Grafana.
- Each dashboard keeps today's scope mapping (section 5.4).
- Losing marker edit/delete for scoped annotations is accepted and documented; PMM-15073 restores it.

## 3. Decision

Grafana stays the record of every annotation. pmm-managed also writes a `pmm_annotation_event` series per scoped node
or service into VictoriaMetrics, and dashboards read scoped annotations from that series.

| Option | Why not |
|---|---|
| VictoriaMetrics only | Nothing left to list, edit or rebuild from; the weakest fit for PMM-15073. |
| PMM datasource plugin filtering Grafana annotations (F) | Needs a second LBAC implementation in Go that must match VictoriaMetrics' matcher semantics; a new plugin in the Grafana fork; untested. |
| ClickHouse table (K) | Fails LBAC: Grafana's ClickHouse user reads `pmm.*` unfiltered. |
| Tag tricks (B, C), Grafana fork patch (G), no scoping (H) | Wrong annotations when filtered, a permanent fork patch capped at 100 results, or no filtering at all (see the options report). |

## 4. Evidence

Reports: [benchmark](https://claude.ai/artifact/F9wrokAErfZJaAiPGzcJVZ), [options](https://claude.ai/artifact/Hj1vR4Xtmrkma7JnnxMubs)
(private, owner shares). Raw data: `results/`, `probe/RESULTS.md`.

| Finding | Status |
|---|---|
| E shows 4,993 of 4,993 annotations on All at 3.07M series; 477 ms through Grafana, 115 ms in VictoriaMetrics; dashboard loads unchanged | Measured (benchmark) |
| An `environment` LBAC role sees only its environment's annotation series | Measured (benchmark) |
| A custom-label LBAC role (`{team="a"}`) sees only series carrying that label; series without it are hidden from that role | Measured (probe P3) |
| A label value over 4,096 bytes or more than 40 labels drops the whole series; the import still returns 204 | Measured (probe P1, single node) |
| `delete_series` hides a series at once; the same or changed labels can be written again and show within 30 s | Measured (probe P2, single node) |
| Grafana's `prometheus` alert-history backend writes `GRAFANA_ALERTS` with every alert label at each evaluation, continuously from firing to resolve | Measured (probe P4) |
| PMM nodes carry no `environment` or `cluster` label; the benchmark gave node annotations both | Code: `Node.UnifiedLabels` in `managed/models/node_model.go`; bench `gen.py` |
| Remote write to `<PMM_VM_URL>/api/v1/write` is vmagent's path in both modes; HA's vmauth routes only it plus the read paths | Code: `managed/services/agents/vmagent.go:128`; `percona-helm-charts` `pmm-ha/templates/vmauth.yaml:33-43` |
| HA runs VictoriaMetrics v1.128.0-cluster, 90-day retention, response cache on, `maxLabelsPerTimeseries=50` | Code: chart values, operator defaults |
| pmm-managed has no Grafana credential; it reads Grafana's PostgreSQL directly where it must | Code: `managed/services/alerting/provisioning_datasource.go`, telemetry |
| The Grafana annotations API applies no LBAC | Code and benchmark |

## 5. Design

### 5.1 Data model

Metric `pmm_annotation_event`. One series per annotation and scoped object. One sample: value and timestamp are both
the annotation time in milliseconds.

Labels:

| Annotation scope | Labels |
|---|---|
| Service | `Node.UnifiedLabels()` merged with `Service.UnifiedLabels()` the way `models.MergeLabels` merges them, without agent labels: `node_name`, `node_id`, `node_type`, `region`, `az`, …, `service_name`, `service_id`, `service_type`, `environment`, `cluster`, `replication_set`, `external_group`, custom labels |
| Node | `Node.UnifiedLabels()` only, the same set `node_exporter` series carry |
| Both | plus `annotation_id` (Grafana's annotation id), `text`, `tags` (user tags joined with `,`) |

Rule: an annotation is visible to exactly the users who can see the metrics it annotates. LBAC needs nothing extra.
A node annotation is therefore hidden from a role filtered on `environment`, as node metrics are today.

Limits, enforced by pmm-managed because VictoriaMetrics drops silently:

- `text` is cut to 4,096 bytes on a UTF-8 rune boundary. Grafana keeps the full text.
- A series over 40 labels (`__name__` included) rejects the request with `InvalidArgument` before anything is written.
  40 is the single-node limit; HA allows 50.

### 5.2 Write path

`ManagementService.AddAnnotation` (`managed/services/management/annotation.go`):

1. Validate names and tags as today; build the label sets; check the limits.
2. Create the Grafana annotation as today. `grafana.Client.CreateAnnotation` returns Grafana's response message today;
   it returns the annotation id instead.
3. Remote-write the series to `<PMM_VM_URL>/api/v1/write` (snappy-compressed protobuf, `prompb` from the
   `github.com/prometheus/prometheus` module already in `go.mod`). The writer lives in `managed/services/victoriametrics`,
   behind a small interface in `management/deps.go`.
4. If step 3 fails, return `Unavailable` saying the annotation was created but is not shown yet; the sync job adds it.

A request with a node and several services writes one series per service and one for the node, as the Grafana tags do
today.

### 5.3 Sync job

One leader-only service (`haService.AddLeaderService`), at startup and every 5 minutes:

1. Read annotations from Grafana's PostgreSQL (`annotation`, `annotation_tag`, `tag`), connecting the same way
   `alerting/provisioning_datasource.go` does: time within the data retention period, `alert_id = 0`, no
   `pmm_annotation` tag, and at least one tag naming an existing node or service. When a name is both a node and a
   service, the `Node Name:` / `Service Name:` suffix pmm-managed appends to the text decides.
2. Read the existing `(annotation_id, node_name, service_name)` triples from VictoriaMetrics with one range query over
   the retention period.
3. Write the missing series with the original time and current inventory labels.

This covers the upgrade backfill (requirement 3), failed writes in step 5.2.4, and annotations scripts create through
Grafana's API with node or service names as tags. Annotations older than retention, or of deleted nodes and services,
are skipped. It never deletes.

### 5.4 Dashboards

Each of the 69 dashboards with "PMM Annotations" gets two entries:

- "PMM Annotations" (Grafana): tags `["pmm_annotation"]` only. Dropping `$node_name` / `$service_name` avoids showing a
  scoped annotation twice when one object is selected.
- A new Prometheus entry, "PMM Annotations: nodes and services", toggled separately:

  ```
  max by (annotation_id, text, tags) (
    last_over_time(pmm_annotation_event{<matchers>}[$__interval] offset -$__interval))
  ```

  with `useValueForTime: true`, `textFormat: "{{text}}"`, `tagKeys: "tags"`. `offset -$__interval` compensates for
  Grafana rounding the range down to the step (`pkg/promlib/models/query.go`); `max by` merges a multi-service
  annotation into one marker. The `by` clause and `tagKeys` differ from the benchmark (`node_name`) and need a check.

Matchers copy each dashboard's panel matchers:

| Dashboard shape | Example | Matchers |
|---|---|---|
| Single-select instance or node | MySQL Instance Summary, Node Summary | `service_name="$service_name"` or `node_name="$node_name"` |
| Service overview, multi-select, All = `.+` | MySQL Instances Overview | `environment`, `cluster`, `replication_set`, `region`, `node_name`, `service_name` variables, `service_type="<type>"` |
| Node overview, multi-select, All = `.+` | Home, Nodes Overview | `node_name=~"$node_name_list"` (carries the environment filter), `region`, `node_type` |

Scope mapping, unchanged from `main`:

| Dashboard tags today | Count | Shows |
|---|---|---|
| `$service_name` | 39 | service annotations: `service_name!=""` |
| `$node_name` | 14 | node annotations: `service_name=""` |
| both | 8 | both, as two `or` terms |
| `pmm_annotation` only | 7 | global only, no new entry |
| `$namespace,$pod` | 1 | unchanged |

Node overviews without a hidden `node_name_list` get one. A script rewrites the definitions, and
`dashboards/misc/test_annotations.py` checks that every dashboard with "PMM Annotations" has the scoped entry with
matchers for its variables and that no Grafana tag query uses `$node_name` or `$service_name`.

### 5.5 HA

- Writes go through vmauth's `/api/v1/write` route; no chart change for this ticket.
- The sync job runs on the leader only and resumes on a new leader.
- vmselect has its response cache on, so a backfilled annotation may stay hidden on cached ranges (inferred). If the
  HA run confirms it, set `-search.disableCache=true` on vmselect in `percona-helm-charts`, as single node does by
  default.
- HA keeps metrics 90 days and single node 30; the sync job uses PMM's data retention setting.

### 5.6 What the linked tickets would add

- PMM-15073, edit and delete: a pmm-managed API updates Grafana, deletes the series by `annotation_id`
  (`delete_series`, probe P2) and writes the new ones. HA needs a vmauth route to vmselect's delete path. On v1.128 each
  delete resets the series-ID cache, a short ingest slowdown (from code). On v1.152 re-writing an identical label set
  into a different month may bring the deleted sample back (from code, untested), so an edit must change the label set.
- PMM-15144, alerts: Grafana with `backend = multiple`, `primary = annotations`, `secondaries = prometheus` and a
  datasource that writes straight to VictoriaMetrics (vmproxy refuses writes), then a dashboard entry on
  `GRAFANA_ALERTS` with the same matchers. A continuous firing series renders as a region (from Grafana's code).

## 6. Behaviour changes for the release note

- Node- and service-scoped annotations show when variables are on All, filtered by the other variables.
- LBAC applies to scoped annotations; today any user can read them through the Grafana API.
- Scoped annotations show for the metric retention period.
- Their markers can no longer be edited or deleted from a dashboard; PMM-15073 tracks a replacement.
- A request whose series would exceed 40 labels is rejected.

## 7. Testing and verification

- Unit (pmm-managed): label sets, limits, rune-safe truncation, the remote-write payload against a mocked endpoint,
  `AddAnnotation` ordering and its error on a failed write, the sync job against `go-sqlmock` (skip rules, existing
  triples skipped), leader-only registration.
- Dashboards: `test_annotations.py`, `cleanup-dash.py`, `test_all_value.py`.
- API tests: series written with the expected labels for a service and a node; `InvalidArgument` over 40 labels.
- Live, single node, Feature Build:
  - the three requirements on Home, Nodes Overview, Node Summary, MySQL Instances Overview, MySQL Instance Summary;
  - LBAC with an `environment` role and a custom-label role;
  - upgrade from the current release with existing scoped annotations;
  - the benchmark at 3M series and 5,000 annotations, including the `node_name_list` matcher at 1,000 nodes;
  - VictoriaMetrics stopped during `AddAnnotation`: an error, then the series within 5 minutes.
- HA on CHAOS k3s: writes through vmauth, LBAC, failover of the sync job, the vmselect cache with backfilled data.

Expected pull requests: `percona/pmm` (pmm-managed, dashboards, api-tests, documentation), `percona-helm-charts` only if
the HA run needs the cache flag, and a Feature Build in `Percona-Lab/pmm-submodules`.

## 8. Risks and open checks

| Item | Plan |
|---|---|
| Cost of `node_name=~"$node_name_list"` at 1,000 nodes | Not measured; benchmark it first |
| vmselect response cache hiding backfills in HA | HA run; chart flag if confirmed |
| Grafana-side deletes leave the series behind | Accepted until PMM-15073 |
| Inventory label changes are not reflected in already-written series | Accepted |
| Whether today's tag query matches dashboard-level annotations as well as org-level ones | Check before writing the sync query |
| The global Grafana query keeps `limit: 100` | Unchanged from `main`; global annotations only |
| Marker text: HTML is sanitized and newlines collapse | Same as native annotations |

## 9. Out of scope

Edit or delete API, the UI scope selector, alert annotations, global annotation storage, and the ClickHouse
datasource ignoring LBAC (separate, security-sensitive ticket per `SECURITY.md`).

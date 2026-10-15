# PMM-15663 Scoped Annotations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show node- and service-scoped annotations on All, under LBAC, with an admin delete API, as designed in `DESIGN.md`.

**Architecture:** Grafana stays the record. pmm-managed writes one `pmm_annotation_event` series per scoped node or
service to VictoriaMetrics by remote write, deletes it through `delete_series`, and a leader-only sync job reconciles
the two every 5 minutes. Dashboards read scoped annotations from the series with the same matchers as their panels.

**Tech Stack:** Go (pmm-managed, pmm-admin, api-tests), protobuf + grpc-gateway, `github.com/prometheus/prometheus/prompb`
and `github.com/golang/snappy` (both already in `go.sum`), `lib/pq`, Python 3 `unittest` for dashboards, Helm.

**Spec:** `dev/pmm-15663-bench/DESIGN.md` on branch `PMM-15663-annotation-bench` (percona/pmm). Evidence:
`dev/pmm-15663-bench/probe/RESULTS.md`, `dev/pmm-15663-bench/results/`.

## Global Constraints

- Branches: `PMM-15663-scoped-annotations` in percona/pmm, created from `origin/PMM-15609-fleet-all-value`
  ([percona/pmm#6045](https://github.com/percona/pmm/pull/6045)), because the dashboards build on its `allValue: ".+"`
  and `node_name_list`; merge `main` into it once #6045 merges. `PMM-15663-vmauth-delete-route` in
  percona-helm-charts, from `main`.
- Commits: `PMM-15663 <summary>` (≤50 chars), `git commit -s`, `git add` and `git commit` as separate calls, no
  Co-Authored-By or "Generated with" lines. Opening PRs and posting on GitHub or Jira need the user's go-ahead.
- Metric `pmm_annotation_event`; reserved labels `annotation_id`, `text`, `tags`; value and timestamp both the
  annotation time in ms.
- `text` at most 4,096 bytes, cut on a UTF-8 rune boundary; a series at most 40 labels with non-empty values,
  `__name__` included. Over 40 → `InvalidArgument` before anything is written.
- Node annotation labels: `Node.UnifiedLabels()` only. Service annotation labels: `models.MergeLabels(node, service, nil)`.
- Remote write to `<PMM_VM_URL>/api/v1/write`; delete via `POST <PMM_VM_URL>/api/v1/admin/tsdb/delete_series`; reads
  via `<PMM_VM_URL>/api/v1/query`. Basic auth from the URL's userinfo (HA's vmauth).
- Sync interval 5 minutes; window = `settings.DataRetention`; Grafana annotations with `alert_id = 0` and `org_id = 1`,
  any `dashboard_id` (Grafana's tag query does not filter on it, `pkg/services/annotations/annotationsimpl/xorm_store.go`).
- Go style per `AGENTS.md`: license header on new files, no named returns, no inline `if err :=`, `status.Error` with
  gRPC codes, logrus `*logrus.Entry`, `t.Context()` in tests, mocks via mockery (`.mockery.yaml`), `make gen` after
  proto or interface changes, `make prepare-pr` before declaring Go work done.

## Review Focus

1. A service or node name containing `:` — Grafana stores tag `db:1` as key `db`, value `1`; the sync job must rejoin
   them to resolve the name. Test in Task 7.
2. Annotation text with a multi-byte character straddling byte 4,096 — must be cut before that character, never inside
   it. Test in Task 2.
3. One request with `node_name` plus services on other nodes — each service series carries its own node's labels, and
   the node series carries only that node's. Test in Task 4.
4. Grafana's database unreachable or a query error during sync — nothing may be deleted. Test in Task 7.
5. A custom label named `text`, `tags` or `annotation_id` — the reserved value wins, the custom one is dropped. Test in
   Task 2.

---

### Task 1: Benchmark the node overview matcher (go/no-go)

The spec's one unmeasured cost (DESIGN §8): `node_name=~"$node_name_list"` at 1,000 nodes.

**Files:**
- Create: `dev/pmm-15663-bench/results/bench-node-list.json` and a short note in `dev/pmm-15663-bench/probe/RESULTS.md`
  (on `PMM-15663-annotation-bench`, not the implementation branch)

- [ ] **Step 1:** Provision a CHAOS VM with `pmm-qa:chaos-docker-provisioning` (`--size L --days 1`), start
  `perconalab/pmm-server:3-dev-latest`, copy `gen.py`, `load.sh`, `ann_load.py` and load the stress fleet
  (`bash load.sh` then `FILLER_ONLY=1 bash load.sh --stress`) and `python3 ann_load.py 0 5000` (E only).
- [ ] **Step 2:** Run, median of 5 through Grafana `/api/ds/query` with the Metrics datasource, 30 days, the query
  `max by (annotation_id, text, tags) (last_over_time(pmm_annotation_event{service_name="",node_name=~"<all 1,000 node names joined with |>"}[$__interval] offset -$__interval))`
  and the same with `node_name=~".+"`. Record Grafana time and `executionTimeMsec`.
- [ ] **Step 3:** Go if the list form is under 1 s through Grafana. Otherwise stop and ask the user (the fallback is
  node overviews filtering node annotations without `node_name_list`, so environment does not narrow them on All).
- [ ] **Step 4:** Tear the VM down (`chaos-vm.sh down`), commit the result file on the bench branch, push.

### Task 2: Series labels and limits

**Files:**
- Create: `managed/services/annotations/event.go`
- Test: `managed/services/annotations/event_test.go`

**Interfaces:**
- Produces:
  - `const MetricName = "pmm_annotation_event"`, `MaxTextBytes = 4096`, `MaxLabels = 40`
  - `var ErrTooManyLabels = errors.New("too many labels")`
  - `type Event struct { AnnotationID int64; Time time.Time; Text string; Tags []string; Labels map[string]string }`
    (`Labels` = the object's labels)
  - `func ObjectLabels(node *models.Node, service *models.Service) (map[string]string, error)` — `service` nil for a
    node annotation
  - `func (e Event) SeriesLabels() map[string]string` — `Labels` + `__name__` + reserved labels (reserved win), `text`
    truncated, `tags` joined with `,`, empty values removed
  - `func (e Event) Validate() error` — wraps `ErrTooManyLabels` with the count when `len(SeriesLabels()) > MaxLabels`

- [ ] **Step 1: Write the failing tests** (`event_test.go`, table-driven, `testify/require`):
  - `TestObjectLabels`: node with custom label `team=a` and `region=eu` → contains `node_name`, `region=eu`, `team=a`,
    no `service_name`; with a service (`environment=prod`, custom `team=b`) → contains `service_name`, `environment=prod`,
    `team=b` (service wins over node for the same key, as `MergeLabels` does), and no `agent_id`/`instance`.
  - `TestSeriesLabels`: `__name__="pmm_annotation_event"`, `annotation_id="42"`, `tags="deploy,db"`; a `Labels` entry
    `text="custom"` loses to the event text; an empty `Tags` yields no `tags` key.
  - `TestTruncateText`: 5,000 × `x` → 4,096 bytes; 4,095 × `x` + `é` → 4,095 bytes and valid UTF-8; 4,096-byte input
    unchanged.
  - `TestValidate`: 37 object labels + 3 → nil; 38 + 3 → `errors.Is(err, ErrTooManyLabels)`.
- [ ] **Step 2:** `cd managed && go test ./services/annotations/ -run 'TestObjectLabels|TestSeriesLabels|TestTruncateText|TestValidate' -v`
  → FAIL (package does not compile).
- [ ] **Step 3:** Implement `event.go`. `ObjectLabels` calls `models.MergeLabels(node, service, nil)`; truncation walks
  back from byte 4,096 with `utf8.RuneStart`.
- [ ] **Step 4:** Same command → PASS.
- [ ] **Step 5:** Commit `PMM-15663 Add annotation series labels`.

### Task 3: VictoriaMetrics store

**Files:**
- Create: `managed/services/annotations/store.go`
- Test: `managed/services/annotations/store_test.go`
- Modify: `go.mod` (`github.com/golang/snappy` becomes direct via `go mod tidy`)

**Interfaces:**
- Consumes: `Event`, `MetricName` (Task 2)
- Produces:
  - `type Key struct { AnnotationID int64; NodeName, ServiceName string }`
  - `func NewStore(base *url.URL) *Store` — `base` is `vmParams.ParsedURL()`
  - `func (s *Store) Write(ctx context.Context, events []Event) error`
  - `func (s *Store) Delete(ctx context.Context, ids []int64) error` — no request for empty `ids`
  - `func (s *Store) Keys(ctx context.Context, window time.Duration) ([]Key, error)`

- [ ] **Step 1: Write the failing tests** against `httptest.NewServer`:
  - `TestStoreWrite`: request path `/prometheus/api/v1/write` for base `http://u:p@host/prometheus/`; headers
    `Content-Encoding: snappy`, `Content-Type: application/x-protobuf`, `X-Prometheus-Remote-Write-Version: 0.1.0`,
    basic auth `u`/`p`; snappy-decoded `prompb.WriteRequest` has one series per event with `SeriesLabels()` sorted by
    name, one sample with value and timestamp = `Time.UnixMilli()`. A 500 response → error containing the status.
  - `TestStoreDelete`: `POST /prometheus/api/v1/admin/tsdb/delete_series`, form `match[]` =
    `pmm_annotation_event{annotation_id=~"7|9"}` for `ids` `{7, 9}`; `Delete(ctx, nil)` sends nothing.
  - `TestStoreKeys`: query parameter is
    `count by (annotation_id, node_name, service_name) (last_over_time(pmm_annotation_event[2592000s]))` for a 30-day
    window; a vector result with `annotation_id="7", node_name="n1"` → `[]Key{{7, "n1", ""}}`.
- [ ] **Step 2:** `cd managed && go test ./services/annotations/ -run TestStore -v` → FAIL.
- [ ] **Step 3:** Implement `store.go` with `net/http`, `prompb`, `snappy.Encode`; strip userinfo from the request URL and
  set basic auth from it; non-2xx → `fmt.Errorf` with status and body.
- [ ] **Step 4:** Same command → PASS; `go mod tidy` leaves only `golang/snappy` moving to direct.
- [ ] **Step 5:** Commit `PMM-15663 Add annotation store for VictoriaMetrics`.

### Task 4: AddAnnotation writes the series and returns the id

**Files:**
- Modify: `api/management/v1/annotation.proto` (`AddAnnotationResponse { int64 annotation_id = 1; }`)
- Modify: `managed/services/grafana/client.go` (`CreateAnnotation` returns the id)
- Modify: `managed/services/management/deps.go`, `service.go`, `annotation.go`
- Modify: every `NewManagementService(` call (17, including tests) and `managed/cmd/pmm-managed/main.go`
- Test: `managed/services/grafana/client_test.go`, `managed/services/management/annotation_test.go`

**Interfaces:**
- Consumes: `annotations.Event`, `annotations.ObjectLabels`, `(*annotations.Store).Write` (Tasks 2–3)
- Produces:
  - `func (c *Client) CreateAnnotation(ctx context.Context, tags []string, from time.Time, text, authorization string) (int64, error)`
  - in `deps.go`: `type annotationStore interface { Write(ctx context.Context, events []annotations.Event) error; Delete(ctx context.Context, ids []int64) error }`
  - `NewManagementService(..., vmClient victoriaMetricsClient, annotationStore annotationStore, internalNodePrefixes []string, haEnabled bool)`

- [ ] **Step 1: Write the failing tests:**
  - `client_test.go` `TestCreateAnnotationReturnsID`: httptest Grafana replies `{"message":"Annotation added","id":17}`
    → `17, nil`.
  - `annotation_test.go` (existing `testdb` setup, add `mockAnnotationStore`):
    - "Service annotation writes series": services `s1` on node `n1`; `Write` receives one `Event` with
      `AnnotationID` from the Grafana mock, `Labels["service_name"]="s1"`, `Labels["node_name"]="n1"`, `Tags` = request
      tags only; response `AnnotationId == 17`.
    - "Node and services on other nodes": `node_name=n1`, `service_names=[s2]` where `s2` is on `n2` → two events: one
      with `node_name=n1` and no `service_name`, one with `service_name=s2`, `node_name=n2`.
    - "Global annotation writes nothing": no names → `Write` not called, tags start with `pmm_annotation`.
    - "Store failure": `Write` returns an error → `codes.Unavailable`, Grafana still called once.
    - "Too many labels": a service with 35 custom labels → `codes.InvalidArgument`, Grafana not called.
- [ ] **Step 2:** `cd managed && go test ./services/grafana/ -run TestCreateAnnotationReturnsID && go test ./services/management/ -run TestAnnotations -v`
  → FAIL.
- [ ] **Step 3:** Proto field, `make gen`; `CreateAnnotation` decodes `id`; `deps.go` interface + mockery config,
  `make gen`; `AddAnnotation` builds events from `models.FindServiceByName` / `FindNodeByID` / `FindNodeByName`, calls
  `Validate` on each before Grafana, uses one `now` for Grafana and events, writes after Grafana; `main.go` passes
  `annotations.NewStore(vmParams.ParsedURL())`; other call sites pass `nil` or the mock.
- [ ] **Step 4:** Same commands → PASS; `cd managed && go build ./...`.
- [ ] **Step 5:** Commit `PMM-15663 Write annotation series on AddAnnotation`.

### Task 5: RemoveAnnotation

**Files:**
- Modify: `api/management/v1/annotation.proto`, `api/management/v1/service.proto`
- Modify: `managed/services/grafana/client.go`, `managed/services/management/deps.go`, `annotation.go`
- Test: `managed/services/grafana/client_test.go`, `managed/services/management/annotation_test.go`

**Interfaces:**
- Consumes: `annotationStore.Delete` (Task 4)
- Produces:
  - proto: `message RemoveAnnotationRequest { int64 annotation_id = 1 [(validate.rules).int64.gt = 0]; }`,
    `message RemoveAnnotationResponse {}`, `rpc RemoveAnnotation` with
    `option (google.api.http) = {delete: "/v1/management/annotations/{annotation_id}"}`, summary "Remove an
    Annotation", description "Removes an annotation and the series that show it on dashboards."
  - `var ErrAnnotationNotFound = errors.New("annotation not found")` in package `grafana`
  - `func (c *Client) DeleteAnnotation(ctx context.Context, id int64, authorization string) error` — Grafana 404 →
    `ErrAnnotationNotFound`
  - `grafanaClient` interface gains `DeleteAnnotation`

- [ ] **Step 1: Write the failing tests:**
  - `TestDeleteAnnotation`: httptest expects `DELETE /api/annotations/17` with the caller's `Authorization`; 200 → nil;
    404 → `errors.Is(err, ErrAnnotationNotFound)`.
  - `annotation_test.go` "Remove": Grafana mock then store `Delete(ctx, []int64{17})` in that order (mock `.Once()`
    with `NotBefore`); unknown id → `codes.NotFound`, store not called; store error → `codes.Unavailable`; no
    `authorization` metadata → `codes.Unauthenticated`.
- [ ] **Step 2:** Run as in Task 4 → FAIL.
- [ ] **Step 3:** Proto, `make gen`, client method, interface + `make gen`, `RemoveAnnotation` in `annotation.go`.
- [ ] **Step 4:** Tests PASS; `make check` clean for `api/` (`buf lint`, `buf breaking`).
- [ ] **Step 5:** Commit `PMM-15663 Add RemoveAnnotation API`.

### Task 6: pmm-admin prints the id

**Files:**
- Modify: `admin/commands/annotation.go`
- Test: `admin/commands/annotation_test.go` (create if absent)

- [ ] **Step 1: Write the failing test** `TestAnnotationResult`: `(&annotationResult{AnnotationID: 17}).String()` ==
  `"Annotation added.\nAnnotation ID: 17\n"` — the first line stays as today for scripts that match it.
- [ ] **Step 2:** `cd admin && go test ./commands/ -run TestAnnotationResult -v` → FAIL.
- [ ] **Step 3:** Add `AnnotationID` to `annotationResult` from the response payload (type per the generated client),
  extend `annotationResultT`.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `PMM-15663 Print annotation id in pmm-admin`.

### Task 7: Sync job

**Files:**
- Create: `managed/services/grafana/database.go` (moved `grafanaDatasourceDSN` + `grafanaDBFallback`, exported)
- Modify: `managed/services/alerting/provisioning.go`, `provisioning_datasource.go` (use the moved function)
- Create: `managed/services/annotations/sync.go`, `managed/services/annotations/deps.go`
- Modify: `managed/cmd/pmm-managed/main.go`
- Test: `managed/services/grafana/database_test.go` (move the existing DSN tests), `managed/services/annotations/sync_test.go`

**Interfaces:**
- Consumes: `Event`, `ObjectLabels`, `Key`, `Store` methods (Tasks 2–3)
- Produces:
  - `func DatabaseDSN(fallbackAddr, sslParams string) (string, error)` in package `grafana`
  - `type syncStore interface { Write(ctx context.Context, events []Event) error; Delete(ctx context.Context, ids []int64) error; Keys(ctx context.Context, window time.Duration) ([]Key, error) }`
  - `func NewSyncer(db *reform.DB, grafanaDB *sql.DB, store syncStore) *Syncer`
  - `func (s *Syncer) Run(ctx context.Context)` — one pass at start, then every 5 minutes until `ctx` is done
  - `func (s *Syncer) syncOnce(ctx context.Context) error`

Grafana read (one query; group rows by `a.id` in Go, a tag is `key` or `key:value` when `value != ''`):

```sql
SELECT a.id, a.epoch, a.text, t.key, t.value
FROM annotation a
JOIN annotation_tag at ON at.annotation_id = a.id
JOIN tag t ON t.id = at.tag_id
WHERE a.org_id = 1 AND a.alert_id = 0 AND a.epoch >= $1
```

Existence check for step 4: `SELECT id FROM annotation WHERE id = ANY($1)`.

- [ ] **Step 1:** Move the DSN code and its tests; `cd managed && go test ./services/grafana/ ./services/alerting/` PASS.
- [ ] **Step 2: Write the failing tests** (`sync_test.go`: `go-sqlmock` for Grafana, `testdb` fixtures for inventory as
  `management/annotation_test.go` does, mockery `mockSyncStore`):
  - "Backfills missing series": Grafana row id 5 tagged `s1` (existing service) and `deploy`; `Keys` returns none →
    `Write` gets one event, `AnnotationID 5`, `Time` = epoch, `Tags ["deploy"]`.
  - "Skips present": `Keys` returns `{5, "n1", "s1"}` → `Write` not called.
  - "Skips global, alert-free and unknown": rows tagged `pmm_annotation`, or only unknown names → nothing written.
  - "Rejoins key:value tags" (Review Focus 1): tag row key `db`, value `1` resolves service `db:1`.
  - "Node and service suffix": a name that is both a node and a service resolves by `Node Name:` / `Service Name:` in
    the text.
  - "Deletes positively absent": `Keys` returns id 9; `SELECT id … ANY` returns no rows → `Delete(ctx, []int64{9})`.
  - "Never deletes on read failure" (Review Focus 4): the existence query errors → `Delete` not called, `syncOnce`
    returns the error.
- [ ] **Step 3:** `cd managed && go test ./services/annotations/ -run TestSync -v` → FAIL.
- [ ] **Step 4:** Implement `sync.go`: window = `models.GetSettings(db.Querier).DataRetention`; inventory from
  `models.FindNodes` / `models.FindServices`; compare per `Key`; `Validate` failures are logged and skipped. In
  `main.go`, open `sql.Open("postgres", grafana.DatabaseDSN(*postgresAddrF, q.Encode()))` and register
  `haService.AddLeaderService(ha.NewContextService("annotationsSync", …))`.
- [ ] **Step 5:** PASS; `cd managed && go build ./...`.
- [ ] **Step 6:** Commit `PMM-15663 Add annotation sync job`.

### Task 8: API tests

**Files:**
- Modify: `api-tests/management/annotation_test.go`

- [ ] **Step 1: Write the tests** (run against a live dev server):
  - Extend "Existing service" and "Existing node": response `AnnotationID` non-empty; within 60 s an admin query to
    `/prometheus/api/v1/query` for `pmm_annotation_event{annotation_id="<id>"}` returns one series with
    `service_name` / `node_name` and the service's `environment`.
  - New "Too many labels": a remote node with 39 custom labels → `400`, `InvalidArgument`.
  - New `TestRemoveAnnotation`: create, remove → 200, the series query returns nothing; remove again → `404`.
- [ ] **Step 2:** `make env-up` (on a CHAOS VM per the runner rules, not this host), `make env-root TARGET=run-managed-ci`,
  `make api-test` filtered to `TestAddAnnotation|TestRemoveAnnotation` → PASS.
- [ ] **Step 3:** Commit `PMM-15663 Add annotation API tests`.

### Task 9: Dashboards

**Files:**
- Create: `dashboards/misc/scoped-annotations.py` (idempotent rewrite tool)
- Create: `dashboards/misc/test_annotations.py`
- Modify: every dashboard under `dashboards/dashboards/` whose "PMM Annotations" tags use `$node_name`, `$service_name`
  or `$service`

Rewrite rules for each such dashboard:
- "PMM Annotations" keeps its entry; `tags` and `target.tags` become `["pmm_annotation"]`.
- Insert after it the entry from `dev/pmm-15663-bench/dashboards/E/` (Prometheus datasource
  `{"type": "prometheus", "uid": "PA58DA793C7250F1B"}`) named "PMM Annotations: nodes and services", with
  `useValueForTime: true`, `titleFormat: "Annotation {{annotation_id}}"`, `textFormat: "{{text}}"`,
  `tagKeys: "tags"` and `expr`/`target.expr` =
  `max by (annotation_id, text, tags) (last_over_time(pmm_annotation_event{<m>}[$__interval] offset -$__interval))`,
  two such terms joined by ` or ` when both scopes apply.
- Service term (old tags had a service variable): `service_name!=""` plus `L=~"$V"` for every fleet variable present:
  `environment`, `cluster`, `replication_set`, `region`, `az`, `node_type`, `node_name`, `service_name` (`$service` maps
  to label `service_name`); plus `service_type="<t>"` when the service variable's query uses a single type's metric
  (`mysql_*` → `mysql`, `pg_*`/`postgresql_*` → `postgresql`, `mongodb_*` → `mongodb`, `redis_*`/`valkey_*` → `valkey`,
  `proxysql_*` → `proxysql`, `haproxy_*` → `haproxy`); none for `__name__=~"a|b"` mixes.
- Node term (old tags had `$node_name`): `service_name=""` plus `region`, `az`, `node_type` variables present; plus
  `node_name=~"$node_name_list"` when `node_name` is multi-select with `allValue` and the dashboard has `environment`
  (add a hidden `node_name_list` like #6045's: the `node_name` query with `node_name=~"$node_name"` added, no
  `allValue`, `includeAll`, `multi`, `hide: 2`), else `node_name=~"$node_name"`.

- [ ] **Step 1: Write the failing test** `test_annotations.py` (same layout as `test_all_value.py`):
  - `test_no_scoped_tags`: no Grafana tags query in any dashboard references `$node_name`, `$service_name` or `$service`.
  - `test_scoped_entry_present`: every dashboard with "PMM Annotations" that `git show origin/main` had with a fleet tag
    has exactly one "PMM Annotations: nodes and services" entry with `useValueForTime` true and the expr shape above.
  - `test_matchers_cover_variables`: each `L=~"$V"` references an existing variable, and every fleet variable on the
    dashboard from the rules above appears in the matching term.
  - `test_node_list_used`: a node term on a dashboard with multi `node_name` + `environment` uses `$node_name_list`.
  - `test_type_specific`: MySQL, PostgreSQL, MongoDB and Valkey instance dashboards carry their `service_type`.
- [ ] **Step 2:** `python3 -m unittest discover -s dashboards/misc -p 'test_annotations.py' -v` → FAIL.
- [ ] **Step 3:** Write `scoped-annotations.py`, run it, then `python3 dashboards/misc/cleanup-dash.py` on each changed
  file.
- [ ] **Step 4:** `python3 -m unittest discover -s dashboards/misc -p 'test_*.py' -v` → all PASS (including
  `test_all_value.py` and `test_cleanup_dash.py`); running the rewrite again changes nothing.
- [ ] **Step 5:** Commit `PMM-15663 Read scoped annotations from VictoriaMetrics`.

### Task 10: HA vmauth delete route

**Files (percona-helm-charts, branch `PMM-15663-vmauth-delete-route`):**
- Modify: `charts/pmm-ha/templates/vmauth.yaml`, `charts/pmm-ha/Chart.yaml` (patch version bump)

- [ ] **Step 1:** Add to `url_map`:
  `src_paths: ["/api/v1/admin/tsdb/delete_series"]`,
  `url_prefix: "http://vmselect-{{ include "pmm.fullname" . }}-vmcluster.{{ .Release.Namespace }}.svc.cluster.local:8481/delete/0/prometheus/"`.
- [ ] **Step 2:** `helm template charts/pmm-ha` renders the route; `helm lint charts/pmm-ha` clean.
- [ ] **Step 3:** Commit `PMM-15663 Route annotation deletes to vmselect`.

### Task 11: Documentation

**Files:**
- Modify: `documentation/docs/use/dashboards-panels/annotate/annotate.md`
- Modify: the release notes page for the target version under `documentation/docs/release-notes/`

- [ ] **Step 1:** Per `documentation/AGENTS.md`, document: scoped annotations on All; LBAC applies; they show for the
  metric retention period; `DELETE /v1/management/annotations/{id}` (admins) and the id in `pmm-admin annotate`
  output and the marker title; no edit or delete from the marker, Editors cannot delete them; requests over 40 labels
  are rejected (DESIGN §6).
- [ ] **Step 2:** `make doc-build` succeeds; read the rendered page with `make doc-build-preview`.
- [ ] **Step 3:** Commit `PMM-15663 Document scoped annotations`.

### Task 12: Feature Build and live verification

- [ ] **Step 1:** `make prepare-pr` clean; push both branches; open a Feature Build with `pmm-dev:pmm-feature-build`
  (after the user's go-ahead) using the pmm branch and, for HA, the chart branch.
- [ ] **Step 2:** Single node, CHAOS VM, FB image (DESIGN §7): the three requirements on Home, Nodes Overview, Node
  Summary, MySQL Instances Overview, MySQL Instance Summary with no request URL over 10 KB at 5,000 services; LBAC with
  an `environment` role and a `{team="a"}` role; upgrade from the current release with existing scoped annotations
  (backfill); VictoriaMetrics stopped during `AddAnnotation` (error, then the series within 5 minutes);
  `RemoveAnnotation` and a Grafana-API delete (gone within 5 minutes); the Admin tooltip on a scoped marker (record
  whether Edit/Delete appear). Evidence into `dev/pmm-15663-bench/probe/RESULTS.md` on the bench branch.
- [ ] **Step 3:** HA on CHAOS k3s with `pmm-qa:chaos-ha-provisioning` and the chart branch: writes and deletes through
  vmauth, LBAC, sync job after leader failover, a backfilled annotation behind the vmselect response cache (if hidden,
  add `-search.disableCache=true` for vmselect to the chart branch and retest), ingestion rate during a delete.
- [ ] **Step 4:** Tear down every VM and cluster; report results to the user before any PR is opened.

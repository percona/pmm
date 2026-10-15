# PMM-15663 handoff

State as of 9 Oct 2026. Start a new session by pointing it at this file.

**Update 14 Oct 2026:** the design is now in [`DESIGN.md`](DESIGN.md) (draft, awaiting the
user's review), with probe evidence in [`probe/RESULTS.md`](probe/RESULTS.md). It supersedes
"E design as tested" and "Next steps" below: Grafana stays the record, pmm-managed writes the
series with the node's or service's full labels (nodes carry no `environment`), a leader-only
sync job backfills and mirrors deletes, admins delete through a new `RemoveAnnotation` API
(HA needs a vmauth delete route in percona-helm-charts), and each dashboard keeps today's scope
mapping. Next step after review:
the `superpowers:writing-plans` skill, then the `node_name_list` benchmark and implementation.

## Goal

Node and service scoped annotations disappear when a fleet variable is set to All,
because [percona/pmm#6045](https://github.com/percona/pmm/pull/6045) (PMM-15609) sets
`allValue: ".+"` and Grafana matches annotations by tag. Pick a storage option that
shows them correctly on All, respects LBAC and scales.

## Decisions already made (with the user)

- Retention: same as metrics (30 days by default) is acceptable.
- Scale targets: 500 scoped annotations over 30 days normally, 5,000 as stress; about
  200k series normally, about 3M as stress; 1,000 nodes, 8,000 services.
- Global annotations (`pmm_annotation` tag) stay in Grafana.
- LBAC is a pass/fail criterion.
- Benchmark dashboards: Home, Nodes Overview, MySQL Instances Overview.
- HA: a separate run, only for the winning option.

## Result

Option E won: a `pmm_annotation_event` series in VictoriaMetrics. The ClickHouse table
(K) was faster but fails LBAC (Grafana's ClickHouse user has `SELECT ON pmm.*` and
nothing filters by role). Full report for the team:
https://claude.ai/artifact/F9wrokAErfZJaAiPGzcJVZ (private, owner shares it).
Earlier options overview: https://claude.ai/artifact/Hj1vR4Xtmrkma7JnnxMubs.

Key numbers at 3.07M series, 5,000 annotations, 30 days, All:
E 477 ms through Grafana / 115 ms in VictoriaMetrics, 4,993 of 4,993 shown,
Home load 7.7 s (PR 7.8 s). Raw data in `results/`.

## E design as tested

- One series per annotation and service. Labels: `annotation_id`, `text`, `tags`,
  `node_name`, `environment`, `cluster`, plus `service_name`, `service_type`,
  `replication_set` for service annotations. Value and timestamp = annotation time (ms).
- Dashboard annotation (Prometheus datasource, `useValueForTime: true`,
  `textFormat: "{{text}}"`, `tagKeys: "node_name"`):

  ```
  max by (annotation_id, text, node_name) (
    last_over_time(pmm_annotation_event{environment=~"$environment",node_name=~"$node_name"}
      [$__interval] offset -$__interval))
  ```

  MySQL Instances Overview adds `cluster`, `replication_set`, `service_name` matchers
  and `service_type="mysql"`. Definitions: `dashboards/E/`.
- `offset -$__interval` is required: Grafana rounds start and end down to the step
  (`pkg/promlib/models/query.go:311` in percona/grafana), which otherwise drops the
  newest annotations at long ranges.
- `max by (annotation_id, …)` merges the per-service series of a multi-service annotation.

## Next steps

1. Team discussion of the report's open questions: should service annotations show on
   node dashboards; should node-only annotations show on service dashboards; delete/edit
   (VictoriaMetrics delete-series API or append-only); migrating existing Grafana
   annotations; annotation text vs the label value length limit (4 KB default, unverified).
2. HA run for E on a CHAOS k3s cluster (`pmm-qa:chaos-ha-provisioning` skill). Write
   through vminsert `http://vminsert-pmm-ha-vmcluster.pmm.svc:8480/insert/0/prometheus/api/v1/import`,
   read through Grafana; check LBAC and failover.
3. Implementation: pmm-managed writes the series in `AddAnnotation`
   (`managed/services/management/annotation.go`), single node via the local
   VictoriaMetrics import, HA via vminsert; then change the 69 dashboards that define
   "PMM Annotations" and add a test like `dashboards/misc/test_all_value.py`.
4. Possibly a separate ticket: the ClickHouse datasource ignores LBAC, which also exposes
   QAN data. Treat as security-sensitive; follow `SECURITY.md`.

## How to rerun the benchmark

All scripts are in this directory. Paths and the host name inside them point at the
torn-down VM `PMM-15663-bench`; update `BASE`/`VM` and the scratchpad paths first.

1. Provision: `chaos-vm.sh up <run> --size L --days 3`
   (`/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh`).
   Never run Docker or PMM on the runner host itself.
2. On the VM: `docker run -d --name pmm-server -p 443:8443 perconalab/pmm-server:3-dev-latest`,
   then `docker exec pmm-server change-admin-password <pw>`.
3. Copy `gen.py`, `load.sh`, `ann_load.py` and `ann-5000.json` (`python3 gen.py annotations 5000`)
   to `/root` on the VM. `bash load.sh` loads the baseline fleet;
   `FILLER_ONLY=1 bash load.sh --stress` adds about 2.8M series (about 19 minutes).
   `python3 ann_load.py 0 500` loads annotations into Grafana, VictoriaMetrics and ClickHouse.
4. On the runner: fetch main and PR versions of the three dashboards into `dash/main`
   and `dash/pr`, then `python3 variants.py && python3 upload.py`.
5. `python3 matrix.py <N> <outdir>` (Chromium via Playwright, about an hour for 104 loads;
   `VARIANTS=E,K` to limit) and `python3 bench.py <label> <out.json>`.
6. Tear the VM down: `chaos-vm.sh down <run>`.

## Gotchas found

- Grafana inserts a custom `allValue` raw even with `:singlequote`; K needed
  `match(col, '^(${var:regex})$')`.
- VictoriaMetrics in PMM runs with `-search.disableCache=true`, `-retentionPeriod=30d`;
  data near the 30-day edge ages out during long runs.
- Fresh imports take a minute to become fully queryable.
- Counting all series with a query times out at 3M; use `/api/v1/status/tsdb`.

## Session rules

- Commits `PMM-XXXX Short summary`, `git commit -s`, no Co-Authored-By or "Generated with" lines.
- Don't commit, push or post on GitHub or Jira without the user's go-ahead.
- The runner's commit hook checks the main checkout's branch: it must be a ticket branch.
  Run `git add` and `git commit` as separate calls.
- Replies: direct, say what is verified and what is assumed, end with a summary table.

## Related branches and PRs

- `PMM-15663-annotation-bench` (this branch): scratch, not for merge.
- `PMM-15609-fleet-all-value`: [percona/pmm#6045](https://github.com/percona/pmm/pull/6045).
- `PMM-15609-screenshots`: PMM-15609 evidence.
- Feature build: [Percona-Lab/pmm-submodules#4611](https://github.com/Percona-Lab/pmm-submodules/pull/4611).
- `PMM-15609-companion-var-proto` should be deleted in the GitHub UI (403 from the runner).

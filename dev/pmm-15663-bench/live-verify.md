# PMM-15663 live verification (single node, CHAOS)

Setup: CHAOS VM `PMM-15663-dev`, PMM dev container (`perconalab/pmm-server:3-dev-container`), pmm-managed and
dashboards hot-swapped from the branch (`make run-managed-ci`, `make reload-dashboards`). Databases registered with
pmm-qa's pmm-framework. Not a Feature Build: packaging, the real upgrade path and HA are not covered here.

Before = be751c4b6 (main + PMM-15609 `allValue: ".+"`, the broken state). After = `PMM-15663-scoped-annotations`.

| ID | Check | Result |
|---|---|---|
| V0 | Before: scoped markers missing on All, present when the object is selected | Reproduced: on All the three overviews query Grafana for tags `pmm_annotation`, `.+` and return only the global annotation; a selected node or service returns its own (`live/V0-before.txt`) |
| V1 | Upgrade: annotations made before get series after the switch (sync backfill) | Pass: at startup the leader's sync wrote 4 series (A1, A2, A3, A5; global A4 skipped); service series carry environment, cluster and the custom label `team=a`, node series node labels only (`live/V1-backfill-series.txt`) |
| V2 | Req 1: on All, scoped markers on Home, Nodes Overview, Node Summary, MySQL Instances Overview, MySQL Instance Summary | Pass: on All, Home and Nodes Overview return global + both node annotations, MySQL Instances Overview global + both nodes + both services; markers drawn on panels (`live/V2-*`). URL size at 5,000 services is the Task 1 benchmark, not re-measured here |
| V3 | Req 2: object selected shows only its own and global annotations | Pass: MySQL Instance Summary, Node Summary, Nodes Overview and Home with one object selected return only its annotation + global. MySQL Instances Overview with one service and Node=All also shows node annotations of all MySQL nodes (both-scope ruling, as main before PMM-15609) (`live/V3-*`) |
| V4 | LBAC, `environment` role | Pass: role `{environment="ps-dev"}` sees only A1 (+ global from Grafana); no node annotations, matching its "No data" node panels (`live/V4-V5-lbac.txt`, `live/V4-envuser-home-panel.png`) |
| V5 | LBAC, custom-label role | Pass: role `{team="a"}` sees only A2 (+ global) (`live/V4-V5-lbac.txt`, `live/V5-teamuser-mysql-summary-panel.png`) |
| V6 | RemoveAnnotation: series and marker gone; NotFound for an unknown id | Pass: `DELETE /v1/management/annotations/7` 200, series 7 and Grafana annotation 7 gone at once; unknown id 404 `Annotation with ID 999999 not found.`; Viewer 403 |
| V7 | Delete through Grafana's API: gone within 5 minutes | Pass: Grafana `DELETE /api/annotations/6` at 16:16:14; the 16:17:53 sync pass logged `Deleted the series of deleted annotations. count=1` and series 6 is gone |
| V8 | VictoriaMetrics stopped during AddAnnotation: error, then series within 5 minutes | Pass: 503 with "The annotation was created (ID 10), but it is not shown on dashboards yet: the sync job will add it"; the 16:22:53 sync pass wrote its series (`Wrote missing annotation series. count=1`); the marker shows at its original time (`live/V6-V8-*`) |
| V9 | Admin tooltip on a scoped marker: Edit/Delete buttons | **Defect (UX, known risk §8):** an Admin's tooltip on a scoped marker shows Edit and Delete; Delete sends `DELETE /api/annotations/undefined`, Grafana answers 403 and shows "You'll need additional permissions… annotations:delete"; nothing is deleted. Cause: `AnnotationTooltip2.tsx:31-32` (percona/grafana) tests `annoId !== 0`, and Prometheus events have no id (`live/V9-*`) |
| V10 | pmm-admin annotate prints the annotation ID | Pass: branch pmm-admin prints `Annotation added.` + `Annotation ID: 7` (also for a global annotation); `--json` prints `{"annotation_id":"9"}` (id as a JSON string); the old pmm-admin still works against the new server and its annotation gets a series |
| V11 | pmm-managed and Grafana logs | Pass: no errors or warnings from `annotations/sync`; Grafana's only annotation errors are the two the tests caused (id 999999, id undefined) |

## Log

- Fresh `pmm-data`; be751c4b6 pmm-managed (`run-managed-ci`) and dashboards (`reload-dashboards`) deployed.
- pmm-framework: `ps=8.4` (node `ps_pmm_8_4_1`, service `ps_pmm_8_4_1_97709`, environment `ps-dev`) and `mysql=8.0`
  (node `mysql_pmm_8_0_1`, service `mysql_pmm_8_0_1_68246`, environment `mysql-dev`); `pdpgsql=17` failed (port 5432 is
  taken by the dev container) and is not needed. Custom label `team=a` set on `mysql_pmm_8_0_1_68246` (ChangeService).
- Annotations on the old build (API answers `{}`, no id): Grafana ids 1 A1 service ps (+ tag `t1`), 2 A2 service mysql,
  3 A3 node ps, 4 A4 global, 5 A5 node mysql.
- Switched to the branch (`sync`, `run-managed-ci`, `reload-dashboards`): log `Starting annotationsSync`, then
  `Wrote missing annotation series. count=4`.
- Access control enabled; roles `env-ps-dev` `{environment="ps-dev"}` and `team-a` `{team="a"}`; Grafana Viewer users
  `envuser` and `teamuser`, one role each.
- pmm-admin from the branch run in client container `ps_pmm_8_4_1` (the server container's own agent holds no admin
  credentials, so annotate there answers Unauthorized, as on main). Annotations: 6 (old pmm-admin), 7, 8 (global), 9.
- V8: VictoriaMetrics stopped 16:19:05, AddAnnotation answered 503 `The annotation was created (ID 10), but it is not
  shown on dashboards yet: the sync job will add it. Cause: … connection refused`; VictoriaMetrics started 16:19:07.
- 16:22:53 sync pass wrote A10. Final PS service dashboard: A1, A10 (scoped) and A4, A8 (global); A6 and A7 gone.

## Not covered by this run

- Feature Build packaging and a real package upgrade from the current release (the upgrade here is a binary swap
  over existing Grafana annotations, which exercises the same backfill path).
- HA (vmauth delete route, sync failover, vmselect cache).
- Scale (3M series, 5,000 annotations, 1,000 nodes): Task 1 benchmark only.
- PostgreSQL and other database dashboards; only MySQL and OS dashboards were opened.
- V9 cause (2026-10-15 20:44 UTC): not a missing permission. Admin holds `annotations:delete` on
  `annotations:type:organization` and `folders:*`. Grafana resolves `DELETE /api/annotations/:id` by parsing the id and
  loading the annotation (`pkg/api/annotations.go:551-567` in percona/grafana); `undefined` fails to parse
  (`invalid scope`), a missing id fails to load, and `accesscontrol/middleware.go:122-136` answers both with the
  generic "You'll need additional permissions … annotations:delete" 403. Reproduced as admin: id `undefined` and id
  `999999` both give that 403. No permission makes a marker without an id deletable. Test annotations 11 (global) and
  12 (service) added for manual checks.

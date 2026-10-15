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
| V6 | RemoveAnnotation: series and marker gone; NotFound for an unknown id | |
| V7 | Delete through Grafana's API: gone within 5 minutes | |
| V8 | VictoriaMetrics stopped during AddAnnotation: error, then series within 5 minutes | |
| V9 | Admin tooltip on a scoped marker: Edit/Delete buttons | |
| V10 | pmm-admin annotate prints the annotation ID | |
| V11 | pmm-managed and Grafana logs | |

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

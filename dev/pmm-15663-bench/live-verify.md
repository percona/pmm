# PMM-15663 live verification (single node, CHAOS)

Setup: CHAOS VM `PMM-15663-dev`, PMM dev container (`perconalab/pmm-server:3-dev-container`), pmm-managed and
dashboards hot-swapped from the branch (`make run-managed-ci`, `make reload-dashboards`). Databases registered with
pmm-qa's pmm-framework. Not a Feature Build: packaging, the real upgrade path and HA are not covered here.

Before = be751c4b6 (main + PMM-15609 `allValue: ".+"`, the broken state). After = `PMM-15663-scoped-annotations`.

| ID | Check | Result |
|---|---|---|
| V0 | Before: scoped markers missing on All, present when the object is selected | Reproduced: on All the three overviews query Grafana for tags `pmm_annotation`, `.+` and return only the global annotation; a selected node or service returns its own (`live/V0-before.txt`) |
| V1 | Upgrade: annotations made before get series after the switch (sync backfill) | |
| V2 | Req 1: on All, scoped markers on Home, Nodes Overview, Node Summary, MySQL Instances Overview, MySQL Instance Summary | |
| V3 | Req 2: object selected shows only its own and global annotations | |
| V4 | LBAC, `environment` role | |
| V5 | LBAC, custom-label role | |
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

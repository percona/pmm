# Technology hubs (PMM-15353 prototype)

A hub is the page a technology's sidebar entry opens: `/pmm-ui/mongodb` for MongoDB. `template/` holds the technology-agnostic page and view model, and `mongodb/` is its first configuration.

## Demo variants

The hub opens on sample data, so every variant shows the same fleet (60 clusters in prod, staging and dev) in a different state. Pick one in the demo controls, or with `?hubState=` in the URL. The three status banner scenarios in the [Figma hub](https://www.figma.com/design/IcsWrmoejelWaTSuWhqnEI/Rethinking-wayfinding-in-PMM?node-id=1-2) map to the first three. A facet with something to do sits on a card with its action, in brackets below; a click anywhere on the card takes it.

| Variant                 | Figma banner scenario | URL                                   | Status banner                                                                                                                                                    |
| ----------------------- | --------------------- | ------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Best case (the default) | Healthy               | `/pmm-ui/mongodb`                     | All 60 clusters are healthy • No alerts firing • Backups on schedule, last 25m ago                                                                               |
| Incomplete setup        | Incomplete            | `/pmm-ui/mongodb?hubState=incomplete` | 12 of 60 clusters degraded [Review 15 issues] · No alerts set up [Set up alerts] · 41 of 60 clusters protected [Set up backups]                                  |
| Unhealthy               | Problematic           | `/pmm-ui/mongodb?hubState=unhealthy`  | 23 of 60 clusters need attention [Review 39 issues] · 17 alerts firing, 4 critical [Open alerts] · 3 backups failed, oldest good backup 1d 8h ago [Open backups] |
| First run               |                       | `/pmm-ui/mongodb?hubState=empty`      | No MongoDB monitored yet                                                                                                                                         |
| Live data               |                       | `/pmm-ui/mongodb?hubState=live`       | The local PMM's own MongoDB                                                                                                                                      |

Operations is on by default; with it off (`hubOps=off`), a row that opens Operations' Fleet opens the Instances overview dashboard instead. A viewer (`hubRole=viewer`) cannot set anything up, so the incomplete setup shows its alerts and backups facets without cards.

## Needs attention

A live list of what is wrong right now, merged from PMM's health monitoring, alert rules, agent connections, backups, real-time queries and Operations for MongoDB. A row appears when a problem starts and goes when it ends; nothing is dismissed. With nothing wrong the section is not there at all, as the banner already says all is well.

Each row is written in this format, which the sample data (`mongodb/fixtures/attention.ts`) doubles as the requirements in: `severity` (critical, error or warning), `title`, `impact`, `location`, `environment`, `since`, `issueCount`, `alertFiring`, `alertService`, `source` (kept, never shown) and `destination` (`label`, `route`, `params`, `kind`: page →, inpage ↓ or external ↗).

- Critical rows first, then errors, then warnings; within one severity, by impact, then longest first. Critical rows always stand alone; errors and warnings of one kind across clusters share a row whose title gives the count.
- The heading counts issues (the sum of `issueCount`), not rows. Five rows show; Show N more opens five more at a time, or the rest at once when no more than seven are left, and a scenario change folds them back. The banner's Review N issues opens them all and scrolls there.
- The meta line shows the location only when the title doesn't name it, the environment only without an environment filter, and the firing alert as its own link to Alerts.
- A row about several clusters opens the Clusters page filtered to them (Affected clusters →).
- Live data turns each finding of the health rules into one row, not yet grouped across clusters, and leaves slower queries to the Queries section.

## Demo controls

The button at the viewport's bottom-right, in the page's opposite color mode (or Shift+D), opens a small panel that stays open beside the page: Scenario (Healthy, Incomplete, Unhealthy), then switches to make the freshness indicator stale and to turn Operations for MongoDB on or off (admins only). Every choice goes into the URL, so any combination can be bookmarked. Escape, a click outside or the button closes it. It renders on the dev server and in builds that set `VITE_PMM_PROTOTYPE=true` (the demo host's), never in a production build. The button and panel are `PrototypeControls` (`src/components/prototype-controls`), for any prototype page's demo settings; the hub passes its own in `template/sections/DemoControls.tsx`.

## Freshness

The header's freshness widget never ticks while the data keeps coming. Live shows the RTA page's interval selector, `Auto-refresh: 30s`, with Grafana's intervals (Off, 5s to 1d) and Refresh beside it, which turns during every refresh and is disabled while it does. Off reads `Auto-refresh: OFF` and never goes stale, since the reader chose not to refresh. Once a source misses two refreshes, the widget becomes `Data paused, last update 4m ago` with a warning and Retry, the only state whose age ticks, and returns to Live on the next good refresh. Hovering shows the exact time of the last update. The interval applies to every scheduled source, but a slow one, such as Query Analytics every 5 minutes, never refreshes faster than its own pace.

## Other switches

These combine with any variant; the demo controls set `hubOps`, `hubFreshness` and `refresh`.

| Param          | Values                                                                          | What it changes                                                                                |
| -------------- | ------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `hubOps`       | `on` (default), `off`, `live`                                                   | Operations for MongoDB; `live` follows the server's setting                                    |
| `hubRole`      | `admin`, `viewer`, `live` (default)                                             | The signed-in role; a viewer gets no Add menu and no set-up links                              |
| `hubFreshness` | `stale`                                                                         | Stops refreshing and shows data 4 minutes old; live or sample data                             |
| `refresh`      | `off`, `5s`, `10s`, `30s` (default), `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `1d` | The auto-refresh interval, as Grafana names it; not a prototype param                          |
| `environment`  | an environment name                                                             | Filters every section to that environment                                                      |
| `patches`      | `behind`                                                                        | Shows only the clusters behind on patches; nothing on the hub sets it since Clusters moved out |

On live data, the switches can only take away what the signed-in user has, never add to it. The `hub*` params are prototype-only and go before anything ships.

## Where things live

- Sample data: `mongodb/fixtures/`: the 60-cluster fleet in `fleet.ts`, the three scenarios in `scenarios.ts` and their Needs attention rows in `attention.ts`, plus `empty.ts`. The 5-cluster estate in `estate.ts` (with `healthy.ts`, `incomplete.ts` and `unhealthy.ts`) is the rule tests' sample.
- Status banner facets (cluster health, alerts, backups) and their actions: `template/hub.status.ts`, drawn with Peak UI's `StatusBanner`, `StatusCard` and `StateIndicator`.
- Page layout: the whole hub, header included, sits in one container up to 1440px wide, centred. The status banner is centred at Peak's page width, 1000px. Each row below is a `HubRow` (`template/sections/HubRow.tsx`): 3fr beside 2fr with a 24px gap, 32px between rows, one column below 1024px of the hub's own width (the sidebar narrows it). Row 1 is Needs attention beside the space kept for Recommendations; row 2 is Queries beside Apps and Dashboards. Clusters and backups have pages of their own: the Clusters page, today's Fleet improved, is a placeholder at `/pmm-ui/mongodb/clusters?clusters=a,b` (`template/ClustersPlaceholderPage.tsx`).
- URL params: `template/hub.params.ts`.

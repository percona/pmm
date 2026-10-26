# Technology hubs (PMM-15353 prototype)

A hub is the page a technology's sidebar entry opens: `/pmm-ui/mongodb` for MongoDB. `template/` holds the technology-agnostic page and view model, and `mongodb/` is its first configuration.

## Demo variants

The hub opens on sample data, so every variant shows the same estate (5 clusters, 21 members) in a different state. Pick one in the demo controls, or with `?hubState=` in the URL. The three status banner scenarios in the [Figma hub](https://www.figma.com/design/IcsWrmoejelWaTSuWhqnEI/Rethinking-wayfinding-in-PMM?node-id=1-2) map to the first three. A facet with something to do sits on a card with its action, in brackets below; a click anywhere on the card takes it.

| Variant                 | Figma banner scenario | URL                                   | Status banner                                                                                                                         |
| ----------------------- | --------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Best case (the default) | Healthy               | `/pmm-ui/mongodb`                     | All 5 clusters are healthy • No alerts firing • Backups on schedule, last 6h ago                                                      |
| Incomplete setup        | Incomplete            | `/pmm-ui/mongodb?hubState=incomplete` | 2 of 5 clusters degraded [Review 2 issues] · No alerts set up [Set up alerts] · 3 of 4 clusters protected [Set up backups]            |
| Unhealthy               | Problematic           | `/pmm-ui/mongodb?hubState=unhealthy`  | 4 of 5 clusters need attention [Review 15 issues] · 3 alerts firing [Open alerts] · 2 backups failed, last good 6h ago [Open backups] |
| First run               |                       | `/pmm-ui/mongodb?hubState=empty`      | No MongoDB monitored yet                                                                                                              |
| Live data               |                       | `/pmm-ui/mongodb?hubState=live`       | The local PMM's own MongoDB                                                                                                           |

Operations is on by default; with it off (`hubOps=off`), the unhealthy scenario reads payments' upgrade as version drift, so all 5 clusters need attention. A viewer (`hubRole=viewer`) cannot set anything up, so the incomplete setup shows its alerts and backups facets without cards.

## Demo controls

The button at the hub's bottom-right, in the page's opposite color mode (or Shift+D), opens a small panel that stays open beside the page: Scenario (Healthy, Incomplete, Unhealthy), then switches to make the freshness indicator stale and to turn Operations for MongoDB on or off (admins only). Every choice goes into the URL, so any combination can be bookmarked. Escape, a click outside or the button closes it. It renders on the dev server and in builds that set `VITE_PMM_PROTOTYPE=true` (the demo host's), never in a production build.

## Freshness

The header's freshness widget never ticks while the data keeps coming. Live shows the RTA page's interval selector, `Auto-refresh: 30s`, with Grafana's intervals (Off, 5s to 1d) and Refresh beside it, which turns during every refresh and is disabled while it does. Off reads `Auto-refresh: OFF` and never goes stale, since the reader chose not to refresh. Once a source misses two refreshes, the widget becomes `Data paused, last update 4m ago` with a warning and Retry, the only state whose age ticks, and returns to Live on the next good refresh. Hovering shows the exact time of the last update. The interval applies to every scheduled source, but a slow one, such as Query Analytics every 5 minutes, never refreshes faster than its own pace.

## Other switches

These combine with any variant; the demo controls set `hubOps`, `hubFreshness` and `refresh`.

| Param          | Values                                                                          | What it changes                                                                        |
| -------------- | ------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `hubOps`       | `on` (default), `off`, `live`                                                   | Operations for MongoDB; `live` follows the server's setting                            |
| `hubRole`      | `admin`, `viewer`, `live` (default)                                             | The signed-in role; a viewer gets no Add menu and no set-up links                      |
| `hubFreshness` | `stale`                                                                         | Stops refreshing and shows data 4 minutes old; live or sample data                     |
| `refresh`      | `off`, `5s`, `10s`, `30s` (default), `1m`, `5m`, `15m`, `30m`, `1h`, `2h`, `1d` | The auto-refresh interval, as Grafana names it; not a prototype param                  |
| `environment`  | an environment name                                                             | Filters every section to that environment                                              |
| `patches`      | `behind`                                                                        | Shows only the clusters behind on patches; the chip above the clusters tree toggles it |

On live data, the switches can only take away what the signed-in user has, never add to it. The `hub*` params are prototype-only and go before anything ships.

## Where things live

- Sample data: `mongodb/fixtures/` (`healthy.ts`, `incomplete.ts`, `unhealthy.ts`, `empty.ts`), all built on the estate in `estate.ts`.
- Status banner facets (cluster health, alerts, backups) and their actions: `template/hub.status.ts`, drawn with Peak UI's `StatusBanner`, `StatusCard` and `StateIndicator`.
- Page layout: each row under the header is a `HubBand` (`template/sections/HubBand.tsx`) with its own max width. The status banner's is Peak's page width, 1000px; the rows below stay full width until their designs set one.
- URL params: `template/hub.params.ts`.

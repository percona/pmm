# Technology hubs (PMM-15353 prototype)

A hub is the page a technology's sidebar entry opens: `/pmm-ui/mongodb` for MongoDB. `template/` holds the technology-agnostic page and view model, and `mongodb/` is its first configuration.

## Demo variants

The hub opens on sample data, so every variant shows the same estate (5 clusters, 21 members) in a different state. Pick one in the Prototype menu at the bottom of the page, or with `?hubState=` in the URL. The three status banner scenarios in the [Figma hub](https://www.figma.com/design/IcsWrmoejelWaTSuWhqnEI/Rethinking-wayfinding-in-PMM?node-id=1-2) map to the first three. A facet with something to do sits on a card with its action, in brackets below; a click anywhere on the card takes it.

| Variant                 | Figma banner scenario | URL                                   | Status banner                                                                                                                         |
| ----------------------- | --------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Best case (the default) | Healthy               | `/pmm-ui/mongodb`                     | All 5 clusters are healthy • No alerts firing • Backups on schedule, last 6h ago                                                      |
| Incomplete setup        | Incomplete            | `/pmm-ui/mongodb?hubState=incomplete` | 2 of 5 clusters degraded [Review 2 issues] · No alerts set up [Set up alerts] · 3 of 4 clusters protected [Set up backups]            |
| Worst case              | Problematic           | `/pmm-ui/mongodb?hubState=wrong`      | 4 of 5 clusters need attention [Review 15 issues] · 3 alerts firing [Open alerts] · 2 backups failed, last good 6h ago [Open backups] |
| First run               |                       | `/pmm-ui/mongodb?hubState=empty`      | No MongoDB monitored yet                                                                                                              |
| Live data               |                       | `/pmm-ui/mongodb?hubState=live`       | The local PMM's own MongoDB                                                                                                           |

Operations is on by default; with it off (`hubOps=off`), the worst case reads payments' upgrade as version drift, so all 5 clusters need attention. A viewer (`hubRole=viewer`) cannot set anything up, so the incomplete setup shows its alerts and backups facets without cards.

## Other switches

These combine with any variant, and the Prototype menu sets them too.

| Param           | Values                              | What it changes                                                                        |
| --------------- | ----------------------------------- | -------------------------------------------------------------------------------------- |
| `hubOps`        | `on` (default), `off`, `live`       | Operations for MongoDB; `live` follows the server's setting                            |
| `hubRole`       | `admin`, `viewer`, `live` (default) | The signed-in role; a viewer gets no Add menu and no set-up links                      |
| `hubConnection` | `lost`                              | Stops sample data refreshing, so the header warns that data looks stale                |
| `environment`   | an environment name                 | Filters every section to that environment                                              |
| `patches`       | `behind`                            | Shows only the clusters behind on patches; the chip above the clusters tree toggles it |

On live data, the switches can only take away what the signed-in user has, never add to it. The `hub*` params are prototype-only and go before anything ships.

## Where things live

- Sample data: `mongodb/fixtures/` (`healthy.ts`, `incomplete.ts`, `wrong.ts`, `empty.ts`), all built on the estate in `estate.ts`.
- Status banner facets (cluster health, alerts, backups) and their actions: `template/hub.status.ts`, drawn with Peak UI's `StatusBanner`, `StatusCard` and `StateIndicator`.
- Page layout: each row under the header is a `HubBand` (`template/sections/HubBand.tsx`) with its own max width. The status banner's is Peak's page width, 1000px; the rows below stay full width until their designs set one.
- URL params: `template/hub.params.ts`.

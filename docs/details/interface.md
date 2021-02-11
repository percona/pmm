# User Interface components

1. [Main menu](#main-menu) (also *Grafana menu*, *side menu*)
2. [Navigation bar](#navigation-bar)
3. [View controls](#view-controls)
4. [View selectors](#view-selectors) (dynamic contents)
5. [Shortcut menu](#shortcut-menu) (dynamic contents)

![](../_images/PMM_Home_Dashboard_TALL_Numbered.png)

## Main menu

The main menu is part of the Grafana framework and is visible on every page.

| Item (Top)          | Name                 | Description
| ------------------- | -------------------- | ------------------------------
| {{icon.percona}}    | Home                 | Link to home dashboard
| {{icon.search}}     | Search               | Search dashboards by name
| {{icon.plus}}       | Create               | Create dashboards or [folders][Folders], import dashboards
| {{icon.apps}}       | Dashboards           | Manage dashboards, create playlists, manage snapshots
| {{icon.compass}}    | Explore              | Run queries with [PromQL][PromQL]
| {{icon.bell}}       | Alerting             | Alerting, [Integrated Alerting](../using/alerting.md), Alert Rules, Notification Channels
| {{icon.cog}}        | Configuration        |
| {{icon.shield}}     | Server Admin         |
| {{icon.database}}   | DBaaS                |

[Folders]: https://grafana.com/docs/grafana/latest/dashboards/dashboard_folders/
[PromQL]: https://grafana.com/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/

!!! alert alert-info "Note"
    The DBaaS icon appears only if a server feature flag has been set.

| Icon (Bottom)            | Description          |
|:------------------------:| ---------            |
| (Profile icon)           | User menu            |
| {{icon.questioncircle}}  | Help                 |

### Navigation bar

![Common page elements top row](../_images/PMM_Home_Dashboard_Menus_Top_Navigation_Bar.jpg)

| Item (left)                   | Description               |
| ----------------------------- | ------------------------- |
| {{icon.apps}}                 | (Display only)            |
| (Name) /                      | (Optional) Folder name    |
| (Name)                        | Dashboard name            |
| {{icon.star}}                 | Mark as favorite          |
| {{icon.share}}                | Share dashboard           |
|                               |                           |

### View controls

| Item (right)                  | Description               |
| ----------------------------- | ------------------------- |
| {{icon.cog}}                  | Dashboard settings        |
| {{icon.monitor}}              | Cycle view mode           |
| {{icon.clock9}} (time range)  | Time range selector       |
| {{icon.searchminus}}          | Time range zoom out       |
| {{icon.sync}}                 | Refresh dashboard         |
| (Time interval)               | Refresh period            |

### View selectors

This menu bar is context sensitive; it changes according to the page you are on. (With wide menus on small screens, items may wrap to the next row.)

![](../_images/PMM_Home_Dashboard_Menus_Submenu_Bar.jpg)

| Item                          | Description                               |
| ----------------------------- | ----------------------------------------  |
| Interval                      | Data interval                             |
| Region                        | Filter by region                          |
| Environment                   | Filter by environment                     |
| Cluster                       | Filter by cluster                         |
| Replication Set               | Filter by replication set                 |
| Node Name                     | Filter by node name                       |
| Service Name                  | Filter by service name                    |
| PMM Annotations               | View [annotations](../how-to/annotate.md) |

### Shortcut menu

This menu contains shortcuts to other dashboards. The list changes according to the page you're on.

| Item                          | Description                      |
| ----------------------------- | -------------------------------- |
| {{icon.filealt}} Home         | Home dashboard                   |
| {{icon.apps}} Query Analytics | Query Analytics                  |
| {{icon.bolt}} Compare         | Nodes compare                    |
| (Service Type)                | Service type menu (see below)    |
| {{icon.bars}} HA              | HA dashboards                    |
| {{icon.bars}} Services        | Services menu                    |
| {{icon.bars}} PMM             | PMM menu                         |

!!! alert alert-info "Note"
    The *Compare* menu links to the Instances Overview dashboard for the current service type.

##### Services menu

The *Services* menu choice determines the Service Type menu.

| Menu      | Item                           | Service type menu        | Description           |
| --------- | ------------------------------ | ------------------------ | --------------------- |
| Services  |                                |                          |                       |
|           | MongoDB Instances Overview     | {{icon.bars}} MongoDB    | MongoDB dashboards    |
|           | MySQL Instances Overview       | {{icon.bars}} MySQL      | MySQL dashboards      |
|           | Nodes Overview                 | {{icon.bars}} OS         | OS dashboards         |
|           | PostgreSQL Instances Overview  | {{icon.bars}} PostgreSQL | PostgreSQL dashboards |

##### PMM menu

This item lists shortcuts to utility pages.

| Menu           | Item                            |
| -------------- | ------------------------------- |
| PMM            |                                 |
|                | PMM Add Instance                |
|                | PMM Database Checks             |
|                | PMM Inventory                   |
|                | PMM Settings                    |

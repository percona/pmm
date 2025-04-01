# UI components

How to log in, how the user interface is laid out, and what the controls do.

PMM's user interface is a browser application based on [Grafana](https://grafana.com/docs/grafana/latest/).

![!image](../../images/PMM_Home_Dashboard_Numbered.png)

## Main menu

The main menu is part of the Grafana framework and is visible on every page.

| Item (Top)                         | Name                 | Description
|:----------------------------------:|----------------------|-------------------------------
| <i class="uil uil-star"></i>       | Starred              | Mark your favorite dashboards.
| <i class="uil uil-apps"></i>       | Dashboards           | Create dashboards or [folders][Folders], manage dashboards, import dashboards, create playlists, manage snapshots.
| <i class="uil uil-compass"></i>    | Explore              | Run queries with [PromQL].
| <i class="uil uil-compass"></i>     | Operating System (OS)    | Operating System dashboard
| :simple-mysql: :simple-mongodb: :simple-postgresql: | Service Type dashboards |   Navigate to the dashboards available for the [services added for monitoring](../../install-pmm/install-pmm-client/connect-database/index.md) (MySQL, MongoDB, PostgreSQL, HAproxy or ProxySQL).
| <i class="uil uil-chart"></i> | Query Analytics (QAN) | Navigate to the Query Analytics dashboard where you can analyze database queries over time, optimize database performance, and identify the source of problems.
| <i class="uil uil-bell"></i>       | Alerting             | [Alerting](../../alert/index.md), Create new alerts and manage your alert rules and alert templates.
| <i class="uil uil-search-alt"></i> |  Advisors  | Run health assessment checks against your connected databases and check any failed checks.
| <i class="uil uil-history"></i>    | Backup     | [Backup management and storage location configuration][BACKUP]. The Backup icon appears when **Backup Management** is activated in :material-cog: **PMM Configuration** > :material-cog-outline: **Settings** > **Advanced Settings**.
| :material-cog:                     | Connections        | Access Grafana's built-in data sources within PMM to seamlessly integrate and visualize data from various systems like Prometheus, MySQL, PostgreSQL, InfluxDB, and Elasticsearch.
| :material-cog:                     | PMM Configuration||  Hosts all PMM-related configuration and inventory options.      | 
| <i class="uil uil-shield"></i>     | Administration        |Hosts all Grafana-related configuration and inventory options.
| <i class="uil uil-cloud"></i>      | Entitlements        |This tab is displayed after connecting PMM to Percona Portal, and shows all your Percona Platform account information. 
| <i class="uil uil-ticket"></i>     | List of tickets opened by Customer Support      | Shows the list of tickets opened across your organization. This tab is only available after you connect PMM to Percona Portal.
| <i class="uil uil-clouds"></i>     | Environment Overview        | This tab is displayed after connecting PMM to Percona Portal. Shows the name and email of the Customer Success Manager assigned to your organization, who can help with any PMM queries. This tab will soon be populated with more useful information about your PMM environment.


!!! info alert alert-info "See also"
    - [How to render dashboard images](../../use/dashboards-panels/share-dashboards/share_dashboard.md#render-panel-image)
    - [How to annotate special events](../../use/dashboards-panels/annotate/annotate.md)

[grafana]: https://grafana.com/docs/grafana/latest/
[promql]: https://prometheus.io/docs/prometheus/latest/querying/basics/

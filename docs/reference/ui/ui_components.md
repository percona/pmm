# UI components

![!image](../../_images/PMM_Home_Dashboard_Numbered.png)


!!! note alert alert-light "Key"
    1. [Main menu](#main-menu) (also *Grafana menu*, *side menu*)
    2. [Navigation bar](#navigation-bar)
    3. [View controls](#view-controls)
    4. [View selectors](#view-selectors) (dynamic contents)
    5. [Shortcut menu](#shortcut-menu) (dynamic contents)

## Main menu

The main menu is part of the Grafana framework and is visible on every page.

| Item (Top)                         | Name                 | Description
|:----------------------------------:|----------------------|-------------------------------
| <i class="uil uil-star"></i>       | Starred              | Mark your favorite dashboards.
| <i class="uil uil-apps"></i>       | Dashboards           | Create dashboards or [folders][Folders], manage dashboards, import dashboards, create playlists, manage snapshots.
| <i class="uil uil-compass"></i>    | Explore              | Run queries with [PromQL].
| ![!image](../../_images/os-dashboard.png)      | Operating System (OS)    | Operating System dashboard
| ![!image](../../_images/mysql-dashboard.png) ![!image](../../_images/mongo-dashboard.png) ![!image](../../_images/haproxy-dashboard.png)  ![!image](../../_images/postresql-dashboard.png)  ![!image](../../_images/qan-dashboard.png)| Service Type dashboards |   Navigate to the dasboards available for the [services added for monitoring](../../install-pmm/install-pmm-client/connect-database/index.md) (MySQL, MongoDB, PostgreSQL, HAproxy or ProxySQL). |
 Query Analytics (QAN) | Query Analytics| Navigate to the Query Analytics dashboard where you can analyze database queries over time, optimize database performance, and identify the source of problems.|
| <i class="uil uil-bell"></i>       | Alerting             | [Alerting](../../alert/index.md), Create new alerts and manage your alert rules and alert templates.
| {{icon.checks}}                    |  Advisors  | Run health assessment checks against your connected databases and check any failed checks. 
| <i class="uil uil-history"></i>    | Backup     | [Backup management and storage location configuration][BACKUP]. The Backup icon appears when **Backup Management** is activated in <i class="uil uil-cog"></i> **PMM Configuration > <i class="uil uil-setting"></i> Settings > Advanced Settings**.
| <i class="uil uil-cog"></i>        | Connections        | Access Grafana's built-in data sources within PMM to seamlessly integrate and visualize data from various systems like Prometheus, MySQL, PostgreSQL, InfluxDB, and Elasticsearch.
| <i class="uil uil-cog"></i>        | PMM Configuration||  Hosts all PMM-related configuration and inventory options.      | 
| <i class="uil uil-shield"></i>     | Administration        |Hosts all Grafana-related configuration and inventory options.
| ![!image](../../_images/entitlements-white.png)       | Entitlements        |This tab is displayed after connecting PMM to Percona Portal, and shows all your Percona Platform account information. 
| ![!image](../../_images/support_tickets_white.png)       | List of tickets opened by Customer Support      | Shows the list of tickets opened across your organization. This tab is only available after you connect PMM to Percona Platform.
| ![!image](../../_images/environment_overview.png)       | Environment Overview        | This tab is displayed after connecting PMM to Percona Portal. Shows the name and email of the Customer Success Manager assigned to your organization, who can help with any PMM queries. This tab will soon be populated with more useful information about your PMM environment.



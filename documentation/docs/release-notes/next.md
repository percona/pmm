# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

### Inventory page now in native PMM UI

PMM has used [its own navigation](3.6.0.md#new-native-pmm-navigation-and-revamped-user-interface) instead of Grafana's menus since 3.6.0, and its pages are gradually moving from Grafana to native PMM UI. [**Inventory**](../use/dashboard-inventory.md) now joins [**Updates**](3.0.0.md#new-upgrade-ui), **Help**, [**Real-Time Analytics**](3.7.0.md#real-time-query-analytics-rta-for-mongodb), [**Settings**](3.8.0.md#settings-page-now-in-native-pmm-ui) and [**Alerts > Status**](3.9.0.md#alert-status-page-view-alerts-by-node-or-service), which already run there.

You get the same services, nodes and agents, now with:

- filtered views that you can bookmark and share, while your existing bookmarks keep working
- a simpler layout, with the details and actions of each service and node right in its row:

    ![Inventory page in native PMM UI](../images/PMM_Inventory_Services.png)

## 🔒 Security updates

## 📈 Improvements

- [PMM-15239](https://perconadev.atlassian.net/browse/PMM-15239): The **Alert Status** page now groups alert severities into four bands: **Critical**, **Error**, **Warning** and **Info**. Each band has its own icon and label, so you can tell severities apart without relying on color. The **Severity** filter and sort work by band, and the alert details show the original level, for example **Reported as Emergency**. Alert labels stay the same. For how levels map to bands, see [Alert rules](../alert/alert_rules.md).
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).
- [PMM-15002](https://perconadev.atlassian.net/browse/PMM-15002): The **Inventory** page now runs in [native PMM UI](#inventory-page-now-in-native-pmm-ui), so it looks and works like the rest of the PMM interface. You can also bookmark and share filtered views of your services and nodes, and your existing Inventory bookmarks keep working.
- [PMM-15481](https://perconadev.atlassian.net/browse/PMM-15481): You can now export services from **Inventory > Services** to a CSV file. Click **Export** to download the selected services, or every service that matches the search and filters. Each row includes the status, the monitoring status, the agents that failed, and the labels of the service. For details, see [Export services to a CSV file](../use/dashboard-inventory.md#export-services-to-a-csv-file).
- [PMM-15699](https://perconadev.atlassian.net/browse/PMM-15699): [Operations for MongoDB](../use/mongodb-operations.md) now acts as the signed-in user when it calls PMM Extensions, so your PMM role applies and configuration changes are recorded under your name. If you use the API for nodes, scans, installs or settings, authenticate with a service account token or a PMM session. Requests with a username and password now get `401 Unauthenticated`.
- [PMM-15825](https://perconadev.atlassian.net/browse/PMM-15825): Viewers and Editors can now check the health of your MongoDB clusters in [Operations for MongoDB](../use/mongodb-operations.md) without the Admin role. They can open every page except the install form, but can't scan nodes, start or abort installs, remove node entries or change settings. Starting a scan now needs the Admin role, as installs already did.
- [PMM-15846](https://perconadev.atlassian.net/browse/PMM-15846): The execution history of PMM Extensions apps, such as **MySQL Backups**, now shows a warning icon beside the status of a finished run whose logs were partly or wholly lost before PMM could store them, so you can spot it without opening the run. Hover over or tab to the icon for details. For more on task logs, see [Monitor backups](../apps/mysql-backup.md#monitor-backups).

## ✅ Fixed issues

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-15801](https://perconadev.atlassian.net/browse/PMM-15801): Fixed an arbiter added to PMM showing an empty member state and **mongod** as its process on **MongoDB > Operations > Fleet**. An arbiter now shows **Arbiter** as its process and, while it's running, **ARBITER** as its member state. To see arbiters in Fleet, [add them to PMM without a username and password](../install-pmm/install-pmm-client/connect-database/mongodb.md#step-3-add-mongodb-service-to-pmm).

- [PMM-15776](https://perconadev.atlassian.net/browse/PMM-15776): Fixed misleading help on **MongoDB > Operations > Settings**. **Connect to MongoDB** now says that scans log in with the `.mongodb_uri` file in the home directory of the user pmm-agent runs as, not with the monitoring credentials PMM holds. The **Automatic node scans** setting is removed: turning it off also blocked **Scan** and **Scan all** on the **Nodes** page, and PMM turned it back on at every restart. For details, see [Operations for MongoDB](../use/mongodb-operations.md#provide-mongodb-credentials-for-scans).

- [PMM-15845](https://perconadev.atlassian.net/browse/PMM-15845): Fixed an issue where forms in PMM Extensions apps checked some rules that involve several fields only after you submitted the form, and then showed the error in a banner without marking a field. These rules are now checked while you fill in the form. For example, in **Apps > MySQL Backups**, choosing a GPG **Encryption format** with neither **Encrypt backup** nor **Encrypt after backup completes** turned on shows the error on both switches, and the backup isn't created until you fix it. For details, see [Encrypt backups](../apps/mysql-backup.md#encrypt-backups).

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

- [PMM-15559](https://perconadev.atlassian.net/browse/PMM-15559): Fixed an issue where the **Netstat: TCP** panel on the [Network Details](../reference/dashboards/dashboard-network-details.md) dashboard showed ever-growing totals instead of current activity. The panel now shows new connections, resets, and retransmits per second, so you can quickly spot spikes. Currently open connections are now shown on a separate scale on the right side of the chart, so you can compare them with new connections, resets, and retransmits.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

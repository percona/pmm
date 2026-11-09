# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

### Inventory pages now in native PMM UI

PMM has used [its own navigation](3.6.0.md#new-native-pmm-navigation-and-revamped-user-interface) instead of Grafana's menus since 3.6.0, and its pages are gradually moving from Grafana to native PMM UI. [**Inventory**](../use/dashboard-inventory.md) and its [**Add service**](../install-pmm/install-pmm-client/connect-database/index.md) pages now join [**Updates**](3.0.0.md#new-upgrade-ui), **Help**, [**Real-Time Analytics**](3.7.0.md#real-time-query-analytics-rta-for-mongodb), [**Settings**](3.8.0.md#settings-page-now-in-native-pmm-ui) and [**Alerts > Status**](3.9.0.md#alert-status-page-view-alerts-by-node-or-service), so they look and work like the rest of PMM.

On the **Inventory** page, you get the same services, nodes and agents, now with:

- filtered views that you can bookmark and share, while your existing bookmarks keep working
- a simpler layout, with the details and actions of each service and node right in its row:

    ![Inventory page in native PMM UI](../images/PMM_Inventory_Services.png)

## 🔒 Security updates

## 📈 Improvements

- [PMM-15239](https://perconadev.atlassian.net/browse/PMM-15239): The **Alert Status** page now groups alert severities into four bands: **Critical**, **Error**, **Warning** and **Info**. Each band has its own icon and label, so you can tell severities apart without relying on color. The **Severity** filter and sort work by band, and the alert details show the original level, for example **Reported as Emergency**. Alert labels stay the same. For how levels map to bands, see [Alert rules](../alert/alert_rules.md).
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).
- [PMM-15003](https://perconadev.atlassian.net/browse/PMM-15003): The **Inventory > Add service** pages now run in [native PMM UI](#inventory-pages-now-in-native-pmm-ui), so adding a service looks and works like the rest of PMM. Your existing links to these pages keep working.

## ✅ Fixed issues

- [PMM-15003](https://perconadev.atlassian.net/browse/PMM-15003): Fixed two issues when you add a service on **Inventory > Add service**. If you chose HTTPS for an external service, PMM still collected its metrics over HTTP. If you added a MongoDB service with TLS, PMM ignored the **TLS certificate password** that you entered. Both services now connect the way you set them up.

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-15776](https://perconadev.atlassian.net/browse/PMM-15776): Fixed misleading help on **MongoDB > Operations > Settings**. **Connect to MongoDB** now says that scans log in with the `.mongodb_uri` file in the home directory of the user pmm-agent runs as, not with the monitoring credentials PMM holds. The **Automatic node scans** setting is removed: turning it off also blocked **Scan** and **Scan all** on the **Nodes** page, and PMM turned it back on at every restart. For details, see [Operations for MongoDB](../use/mongodb-operations.md#provide-mongodb-credentials-for-scans).

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

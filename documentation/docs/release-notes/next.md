# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15239](https://perconadev.atlassian.net/browse/PMM-15239): The **Alert Status** page now groups alert severities into four bands: **Critical**, **Error**, **Warning** and **Info**. Each band has its own icon and label, so you can tell severities apart without relying on color. The **Severity** filter and sort work by band, and the alert details show the original level, for example **Reported as Emergency**. Alert labels stay the same. For how levels map to bands, see [Alert rules](../alert/alert_rules.md).
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).
- [PMM-15530](https://perconadev.atlassian.net/browse/PMM-15530): Every tool on the [MCP endpoint](../api/mcp.md) now returns structured JSON next to its text, with a link to the same data in PMM and the time window it covers. Errors carry a code and say whether a retry can help. Long lists of services, variables and queries are paged with `limit` and `offset`. The new `pmm_catalog` tool lists the Query Analytics columns that `pmm_top_queries` accepts.
- [PMM-15529](https://perconadev.atlassian.net/browse/PMM-15529): On the [MCP endpoint](../api/mcp.md), `pmm_get_explain` and `pmm_get_schema` now work from a `query_ref` or from a `queryid` alone, and stay within your services. A `query_ref` keeps the time window, so you can explain a query that you saw earlier. `pmm_get_schema` finds the tables of a MySQL query in a live plan. `pmm_query_detail` now reads only your services. `pmm_get_explain` no longer takes the `query` argument: for MongoDB, it explains the example that Query Analytics stored for the query. A `pmm_get_explain` call with `service_id` and `queryid` now searches only the last hour, unless it passes `period_from` and `period_to` or a `query_ref`. Before, such a call found MySQL and PostgreSQL queries of any age, and MongoDB queries of the last 24 hours. To explain an older query, pass `period_from` and `period_to` covering when it ran, or its `query_ref`. Live-inspection calls on one service are limited to one at a time per PMM Server replica: `pmm_get_schema`, and `pmm_get_explain` for MySQL and MongoDB.

## ✅ Fixed issues

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-15776](https://perconadev.atlassian.net/browse/PMM-15776): Fixed misleading help on **MongoDB > Operations > Settings**. **Connect to MongoDB** now says that scans log in with the `.mongodb_uri` file in the home directory of the user pmm-agent runs as, not with the monitoring credentials PMM holds. The **Automatic node scans** setting is removed: turning it off also blocked **Scan** and **Scan all** on the **Nodes** page, and PMM turned it back on at every restart. For details, see [Operations for MongoDB](../use/mongodb-operations.md#provide-mongodb-credentials-for-scans).

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

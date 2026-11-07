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
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): The new [Amazon RDS overview](../use/amazon-rds-overview.md) page, under **Amazon RDS > Overview**, lists the Amazon RDS and Aurora databases that PMM monitors, grouped by Aurora cluster, Multi-AZ DB cluster or replication group. For each group and instance, it shows the role, status, firing alerts, throughput and storage, and links to the instance dashboards. If you attached RDS instances to a PMM Client on another host, upgrade that PMM Client so that PMM gets their AWS details.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): [MySQL Amazon Aurora Details](../reference/dashboards/dashboard-mysql-amazon-aurora-details.md) now works on Aurora MySQL 3 without URL variables, and has new rows for cache and local storage (including the cluster volume used), CloudWatch instance resources, replication, write forwarding, parallel query and Serverless capacity.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): The new [PostgreSQL Amazon Aurora Details](../reference/dashboards/dashboard-postgresql-amazon-aurora-details.md) dashboard shows one Aurora PostgreSQL instance: commits, statement throughput and latency, deadlocks, transaction ID age, cache, CloudWatch instance resources and the lag of each reader. The **Amazon RDS** page now opens it for Aurora PostgreSQL instances. PMM also stops requesting CloudWatch metrics that AWS publishes only for MySQL engines for PostgreSQL instances, which lowers CloudWatch requests by 19 per collection for each PostgreSQL instance. If a PMM Client on another host monitors your Aurora PostgreSQL instances, upgrade it to get the statement and reader lag graphs.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): **CPU Saturation and Max Core Usage** panels now show data for Amazon RDS and Aurora nodes that have Enhanced Monitoring turned on.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): PMM now stores the CloudWatch `FreeableMemory` metric of Amazon RDS and Aurora instances as `aws_rds_freeable_memory_average`. It was stored as `node_memory_Cached_bytes`, which Enhanced Monitoring also uses for the OS cache. If your custom dashboards or alert rules read `node_memory_Cached_bytes` of RDS nodes as freeable memory, change them to `aws_rds_freeable_memory_average`. Data collected before the upgrade keeps the old name.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): Metrics of Amazon RDS and Aurora nodes now carry the `environment`, `cluster` and `replication_set` labels of the service on the node, when the node's services agree on them. Users whose access role filters on these labels now see these metrics, so the Amazon RDS page and the OS dashboards of RDS nodes show data for them. Because the labels change, graphs of an RDS node can show its metrics twice for a few minutes after the upgrade, or after you change these labels of its service.
- [PMM-15868](https://perconadev.atlassian.net/browse/PMM-15868): Filesystem metrics of Amazon RDS and Aurora nodes (`node_filesystem_*`) now have an empty `fstype` label, so **Disk Space** panels show their storage. If you filter RDS filesystems by `fstype` in custom dashboards or alert rules, update the filter.

## ✅ Fixed issues

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-15776](https://perconadev.atlassian.net/browse/PMM-15776): Fixed misleading help on **MongoDB > Operations > Settings**. **Connect to MongoDB** now says that scans log in with the `.mongodb_uri` file in the home directory of the user pmm-agent runs as, not with the monitoring credentials PMM holds. The **Automatic node scans** setting is removed: turning it off also blocked **Scan** and **Scan all** on the **Nodes** page, and PMM turned it back on at every restart. For details, see [Operations for MongoDB](../use/mongodb-operations.md#provide-mongodb-credentials-for-scans).

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

- [PMM-6655](https://perconadev.atlassian.net/browse/PMM-6655): Fixed gaps in the **Amazon Aurora Statement Latency** graph on [MySQL Amazon Aurora Details](../reference/dashboards/dashboard-mysql-amazon-aurora-details.md). A statement type that the instance didn't run during an interval now shows a latency of 0.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

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

## ✅ Fixed issues

- [PMM-15785](https://perconadev.atlassian.net/browse/PMM-15785): Fixed an issue in PMM HA where ClickHouse replicas could end up with different copies of the Query Analytics database, so QAN showed different data depending on the PMM node that served it. PMM now creates the database with one ID on every replica, and the HA leader checks every 10 minutes and recreates any replica that is missing the database or holds a different copy, keeping the copy with the most data. Rows that exist only in a recreated copy are lost. For how to check your install, see [Troubleshoot PMM HA](../troubleshoot/ha_issues.md#query-analytics-shows-different-data-on-different-pmm-nodes).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

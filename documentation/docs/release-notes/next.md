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

- [PMM-15785](https://perconadev.atlassian.net/browse/PMM-15785): Fixed an issue in PMM High Availability setups where the [PMM Query Analytics](../use/qan/index.md) dashboard could miss queries or show changing totals. This could happen after installation, a pod restart, or a restore, when the ClickHouse servers that store QAN data ended up with mismatched copies of the data. PMM now checks the copies every 10 minutes and keeps the one with the most data. Data that exists only in the other copies is deleted. You can [check whether your installation is affected](../troubleshoot/ha_issues.md#query-analytics-shows-different-data-on-different-pmm-nodes) and see how much data PMM removed.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

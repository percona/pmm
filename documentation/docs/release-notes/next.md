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

- [PMM-15919](https://perconadev.atlassian.net/browse/PMM-15919): Fixed the PMM UI becoming unavailable after upgrading from PMM 3.9.1 or earlier with your own Grafana database password in `GF_DATABASE_PASSWORD`, which the upgrade reset to the default. PMM now also creates its built-in alert rules when Grafana uses an external PostgreSQL database without TLS through `GF_DATABASE_URL`. PMM Server no longer records its database passwords in `pg_stat_statements` either, where Query Analytics for PMM Server's own PostgreSQL database could show them.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

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

- [PMM-15339](https://perconadev.atlassian.net/browse/PMM-15339): Starting with PMM 3.8.0, enabling Grafana anonymous access also gave visitors without credentials Viewer access to PMM's own APIs, such as Query Analytics. PMM now refuses those requests, so visitors who aren't signed in can view Grafana dashboards but not PMM features. If Grafana anonymous access is enabled, `/srv/logs/pmm-managed.log` shows a warning at startup. PMM doesn't fully support anonymous access, and support for it will be deprecated, so consider disabling it. For details, see [Supported external variables](../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#supported-external-variables).
- [PMM-15339](https://perconadev.atlassian.net/browse/PMM-15339): If access control is enabled, service account tokens are denied again on the Query Analytics and VictoriaMetrics query APIs, as they were before PMM 3.8.0. If your automation queries these APIs with a service account token, switch it to a user account that has the matching access role. For details, see [About label based access control (LBAC) in PMM](../admin/roles/access-control/intro.md).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

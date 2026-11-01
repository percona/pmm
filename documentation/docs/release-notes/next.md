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

- [PMM-15339](https://perconadev.atlassian.net/browse/PMM-15339): Starting with PMM 3.8.0, enabling Grafana anonymous access also opened PMM's own APIs, such as Query Analytics. Visitors without credentials got Viewer access to them. PMM now refuses those requests. Visitors who aren't signed in can still view Grafana dashboards, but not PMM features. If Grafana anonymous access is enabled, `/srv/logs/pmm-managed.log` shows a warning at startup. PMM doesn't fully support anonymous access, and support for it will be deprecated. Consider disabling anonymous access. For details, see [Supported external variables](../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#supported-external-variables).
- [PMM-15339](https://perconadev.atlassian.net/browse/PMM-15339): If access control is enabled, PMM denies service account tokens on the Query Analytics and VictoriaMetrics query APIs again. Releases before PMM 3.8.0 denied them too. If your automation reads these APIs with a service account token, switch to a user account. Give that account the matching access role. For details, see [About label based access control (LBAC) in PMM](../admin/roles/access-control/intro.md).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

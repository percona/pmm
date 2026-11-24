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

- [PMM-15773](https://perconadev.atlassian.net/browse/PMM-15773): Fixed an issue where a node could stop sending metrics when another `vmagent` process held the metrics queue of PMM Client. PMM Client now stops a `vmagent` that was left running from before it started. If another process takes the queue later, PMM Client logs which process holds it. PMM Server load no longer rises while a `vmagent` keeps failing to start.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

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

- [PMM-15787](https://perconadev.atlassian.net/browse/PMM-15787): Fixed an issue where a PMM High Availability (HA) cluster could stay without a leader for several minutes with two of its three nodes running, after it lost one node and then the leader. A node that fails now keeps its vote, so any two running nodes elect a leader. The `/v1/ha/nodes` endpoint now lists a node that is down with the status `dead` instead of leaving it out. For details, see [Troubleshoot PMM HA Cluster issues](../troubleshoot/ha_issues.md).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

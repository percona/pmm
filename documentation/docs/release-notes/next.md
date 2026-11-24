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

- [PMM-15796](https://perconadev.atlassian.net/browse/PMM-15796): **MongoDB > Operations** now checks every precondition it can before an install starts, and reports every problem at once instead of one at a time. The **Nodes** page and the install form block a node whose scans are failing, whose last scan couldn't reach Percona's repository, or that has no package manager. The install form refuses a replica set name already used in the same environment and cluster, and nodes that run different Linux distributions. With the default data path, it blocks a node whose last scan found less than 5 GiB free. Before the install changes anything on the nodes, its pre-check also makes sure that every node resolves the name of every member and, on Ubuntu, that `curl` is installed. **Install MongoDB** on the **Automations** page now opens the **Nodes** page filtered to **Ready to install**. For details, see [Operations for MongoDB](../use/mongodb-operations.md#check-nodes).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

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

- [PMM-15828](https://perconadev.atlassian.net/browse/PMM-15828): Fixed several MongoDB dashboard panels that showed wrong values or ignored the **Environment** and **Cluster** filters. In [MongoDB Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md), **Chunk distribution** no longer under-reports shards when the cluster has more than one `mongos` router, **Chunks Move Events** and **Chunks Split Events** show the number of events in the last 10 minutes, and **MongoDB Versions** lists only members of the selected cluster. In [MongoDB ReplSet Summary](../reference/dashboards/dashboard-mongodb-replset-summary.md), **Query Efficiency** and **DBs** show correct values. Collection panels on both dashboards now exclude the `admin` and `config` databases. In [MongoDB Instances Overview](../reference/dashboards/dashboard-mongodb-instances-overview.md), **Availability** shows 0% instead of no data for a replica set that never had a primary, and **Time Since Election** follows the dashboard filters. [MongoDB MMAPv1 Details](../reference/dashboards/dashboard-mongodb-mmapv1-details.md) now lists MMAPv1 services in its service selector.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

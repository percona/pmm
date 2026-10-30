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

- [PMM-13674](https://perconadev.atlassian.net/browse/PMM-13674): New Kubernetes dashboards replace the **Kubernetes (experimental)** folder. They show your Kubernetes clusters, the database clusters that run in them from any operator, Helm chart or StatefulSet (MySQL, MongoDB, PostgreSQL and Valkey), and every object that needs attention, under a new **Kubernetes** menu. PMM HA reports its own cluster automatically. To monitor a cluster from PMM Server in Docker or Podman, install the collector described in [Monitor Kubernetes](../reference/dashboards/kubernetes_monitor_operators.md). The **Kubernetes Cluster Overview** dashboard is replaced by **Kubernetes Cluster Summary**, and **Databases on Kubernetes - Summary** by **Database Clusters on Kubernetes**: update bookmarks to the old dashboards.

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

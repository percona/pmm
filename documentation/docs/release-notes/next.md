# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15965](https://perconadev.atlassian.net/browse/PMM-15965): Fixed an issue where the replica set filter on the [MongoDB Collections Overview](../reference/dashboards/dashboard-mongodb-collection_overview.md), [MongoDB Unused Indexes](../reference/dashboards/dashboard-mongodb-unused-indexes.md) and [MongoDB Oplog Details](../reference/dashboards/dashboard-mongodb-oplog-details.md) dashboards could list replica sets from other environments. The **Replication Set** and **Replica Set** filters now list only the replica sets in the selected **Environment**, so choosing one no longer leaves the dashboard empty.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

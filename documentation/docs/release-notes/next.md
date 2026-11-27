# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-13270](https://perconadev.atlassian.net/browse/PMM-13270): Added the [**MongoDB sharded cluster has no active mongos**](../alert/templates_list.md#mongodb_alerts) alert template. It raises a critical alert when none of the mongos routers of a sharded cluster is responding, so you know when applications can no longer reach the cluster, whether the mongos processes are down or their hosts are unreachable. You get one alert per cluster and environment. The template counts only mongos services that have a cluster name, so [add each mongos with the same `--cluster` value](../install-pmm/install-pmm-client/connect-database/mongodb.md).

## ✅ Fixed issues

- [PMM-15773](https://perconadev.atlassian.net/browse/PMM-15773): Fixed an issue where a node could stop sending metrics when another `vmagent` process held the metrics queue of PMM Client. PMM Client now stops a `vmagent` that was left running from before it started. If another process takes the queue later, PMM Client logs which process holds it. PMM Server load no longer rises while a `vmagent` keeps failing to start.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

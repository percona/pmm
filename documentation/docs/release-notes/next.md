# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15773](https://perconadev.atlassian.net/browse/PMM-15773): Fixed an issue where a node could stop sending metrics when another `vmagent` process held the metrics queue of PMM Client. PMM Client now stops a `vmagent` that was left running from before it started. If another process takes the queue later, PMM Client logs which process holds it. PMM Server load no longer rises while a `vmagent` keeps failing to start.

- [PMM-15816](https://perconadev.atlassian.net/browse/PMM-15816): Fixed an issue where PMM Server failed to start when `PMM_DATA_RETENTION` was set in days, such as `30d`. You can now [set data retention](../install-pmm/install-pmm-server/deployment-options/docker/env_var.md) in days or hours, such as `30d` or `720h`.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

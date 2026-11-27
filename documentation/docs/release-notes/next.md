# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15773](https://perconadev.atlassian.net/browse/PMM-15773): Fixed an issue where a node could stop sending metrics when another `vmagent` process held the metrics queue of PMM Client. PMM Client now stops a `vmagent` that was left running from before it started. If another process takes the queue later, PMM Client logs which process holds it. PMM Server load no longer rises while a `vmagent` keeps failing to start.

- [PMM-15364](https://perconadev.atlassian.net/browse/PMM-15364): Fixed an issue where, on a PMM Server without internet access, the Home dashboard took up to a minute to load and PMM showed **Internal server error.** The Home dashboard now loads right away. The update check stops after 10 seconds, and PMM shows **Failed to check for PMM Server updates.** To skip the check in an air-gapped environment, turn off [Check for updates](../configure-pmm/advanced_settings.md#check-for-updates).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

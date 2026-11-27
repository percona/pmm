# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15974](https://perconadev.atlassian.net/browse/PMM-15974): Scans in [Operations for MongoDB](../use/mongodb-operations.md#how-scans-connect-to-mongodb) no longer need MongoDB credentials, and they no longer read the `/root/.mongodb_uri` file. If you created that file only for scans, you no longer need it. With authentication enabled, MongoDB 8.1 and later answers the version command only after authentication, so scans of those servers don't report the running version.

## ✅ Fixed issues

- [PMM-15773](https://perconadev.atlassian.net/browse/PMM-15773): Fixed an issue where a node could stop sending metrics when another `vmagent` process held the metrics queue of PMM Client. PMM Client now stops a `vmagent` that was left running from before it started. If another process takes the queue later, PMM Client logs which process holds it. PMM Server load no longer rises while a `vmagent` keeps failing to start.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

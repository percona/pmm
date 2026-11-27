# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15923](https://perconadev.atlassian.net/browse/PMM-15923): Fixed several issues in **MongoDB > Operations**. Search on **Nodes** and on the **Services** tab of **Fleet** now matches only the names, addresses and endpoints shown, so a search no longer returns unrelated rows. Scanning one node no longer disables **Scan** on every other node. A failed install now says why, under the install summary and when you hover over its status. A new scan no longer collapses a scan group that you have expanded on the **Scans** tab. An install on an Ubuntu node no longer leaves the downloaded packages on the node. For details, see [Operations for MongoDB](../use/mongodb-operations.md).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

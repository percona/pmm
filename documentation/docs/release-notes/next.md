# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15905](https://perconadev.atlassian.net/browse/PMM-15905): Fixed an issue where a table that remembers your hidden columns, such as those on the **Inventory**, **Advisors** and **Real-Time Analytics** pages, could hide them again the next time you opened the page after its column visibility was reset.

- [PMM-15905](https://perconadev.atlassian.net/browse/PMM-15905): Fixed an issue where changing the start or end of a date range filter that was already set on **Alerts > Status** had no effect.

- [PMM-15905](https://perconadev.atlassian.net/browse/PMM-15905): Fixed an issue where clearing the search in an **Inventory** table added `q=undefined` to the page address, so after a reload or from a shared link the table searched for "undefined" and showed no rows.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

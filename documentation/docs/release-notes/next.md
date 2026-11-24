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

- [PMM-15922](https://perconadev.atlassian.net/browse/PMM-15922): Fixed wrong values in the **Repl. lag**, **Conn. free** and **CPU** columns on the **Services** tab of **MongoDB > Operations > Fleet**. A primary showed the replication lag of its slowest secondary, and each secondary showed the worst lag of any member. Each member now shows only its own lag, and a primary shows none. After a primary and a secondary switched roles, **Conn. free** mixed figures from before and after the switch for up to a day, and **CPU** dropped below zero for about 30 seconds. Right after a switch, **Conn. free** now matches what MongoDB reports, and **CPU** follows the member's new load from its next scrape. With the **Rare** metrics resolution, **CPU** was blank in about a third of the refreshes. It now shows a value at every metrics resolution. The **Max repl. lag**, **Min conn. free** and **Max CPU** columns on the **Clusters** tab are corrected too. For details, see [Operations for MongoDB](../use/mongodb-operations.md#view-fleet-health).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

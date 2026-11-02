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

- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).

- [PMM-15806](https://perconadev.atlassian.net/browse/PMM-15806): You can now search the **Clusters** tab on **MongoDB > Operations > Fleet** by cluster name and filter it by **Health**, **Type**, and **Environment**. The filters are saved in the page URL, so you can share a filtered view and use **Back** and **Forward** to move between filters. Each cluster shows its type next to its name. The column chooser also adds optional **Type**, **Last seen**, **Max CPU**, and **Min conn. free** columns. For details, see [Operations for MongoDB](../use/mongodb-operations.md#view-fleet-health).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

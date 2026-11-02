# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

### Real-time Query Analytics (RTA) for MySQL

In addition to MongoDB, RTA now works with MySQL, Percona Server for MySQL and MariaDB services. You get the same live view of running queries you already use for MongoDB, with pause, filters and CSV export.

For MySQL, RTA also shows what's holding a statement up:

- See which statements are waiting for a lock, which transaction holds that lock and how long they've been waiting, so you can end the transaction at the head of the chain.

- Spot inefficient queries by their row counts and full table scans while they're still running.

RTA for MySQL needs PMM Client 3.11.0 or later on the database host. 

To get started, go to **Query Analytics > Real-time** and [start a session for a MySQL service](../use/qan/QAN-realtime-analytics-mysql.md#start-an-rta-session). 

For programmatic access, use the [Real-time Analytics API](https://percona-pmm.readme.io/reference/real-time-analytics).

## 🔒 Security updates

## 📈 Improvements

- [PMM-15283](https://perconadev.atlassian.net/browse/PMM-15283): [Real-time Query Analytics](../use/qan/QAN-realtime-analytics-mysql.md) now works with MySQL, Percona Server for MySQL and MariaDB services, so you can see the statements running right now and find the transaction that's holding a lock they're waiting for.
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

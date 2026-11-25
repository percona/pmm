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

In addition to MongoDB, RTA now works with MySQL, Percona Server for MySQL and MariaDB services. You get the same live view of running queries, with pause, filters and CSV export, so you can see what's happening on your database while a problem is still going on.

For MySQL, RTA also shows:

- statements waiting for locks, the transactions holding those locks and how long they've been waiting, so you can end the transaction at the head of the chain
- inefficient queries, based on row counts and full table scans, while they're still running

RTA for MySQL needs PMM Client 3.12.0 or later on the database host, and you can also control it through the [Real-time Analytics API](https://percona-pmm.readme.io/reference/real-time-analytics). To get started, go to **Query Analytics > Real-time** and [start a session for a MySQL service](../use/qan/QAN-realtime-analytics-mysql.md#start-an-rta-session).

## 🔒 Security updates

## 📈 Improvements

- [PMM-15283](https://perconadev.atlassian.net/browse/PMM-15283): [Real-time Query Analytics](../use/qan/QAN-realtime-analytics-mysql.md) now works with MySQL, Percona Server for MySQL and MariaDB services, so you can see the statements running right now and find the transaction that's holding a lock they're waiting for. For MongoDB services, RTA now shows elapsed times under 10 seconds with millisecond precision, for example `1.500 s`, and the sessions list has a new **Technology** column.

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)

# QAN Stored metrics

Stored metrics captures queries after they complete, so you can review historical performance, find slow queries, and track optimization progress over time.

## Supported databases

Stored metrics supports MySQL, MongoDB, and PostgreSQL with the following requirements:

=== "MySQL"
    - MySQL 5.1 or later (if using the slow query log)
    - MySQL 5.6.9 or later (if using Performance Schema)
    - Percona Server 5.6+ (all Performance Schema and slow log features)
    - MariaDB 5.2+ (for user statistics), 10.0+ (for Performance Schema)

    **Slow query log requirements**

    - Slow query log enabled and configured to write to a file
    - PMM agent has read permissions to the slow query log file on the host
    - MySQL monitoring user needs `SELECT` to read the log file path; `RELOAD` is required only for automatic log rotation

    **Performance Schema requirements**

    - Performance Schema enabled and configured
    - MySQL monitoring user needs `SELECT` to read Performance Schema tables (no log file access required)

    Some limitations and tuning options apply when using MySQL's Performance Schema. See [Query Analytics with MySQL](mysql.md#missing-query-examples-in-mysql-performance-schema).

=== "PostgreSQL"
    - PostgreSQL 11 or later
    - `pg_stat_monitor` extension (recommended) or `pg_stat_statements` extension
    - Appropriate `shared_preload_libraries` configuration
    - Superuser privileges for PMM monitoring account

=== "MongoDB"
    - MongoDB 6.0 or later (4.4+ may work with limited features)

    **Profiler requirements**

    - Profiling enabled for Query Analytics
    - Appropriate user roles: `clusterMonitor`, `read` (local), and custom monitoring roles with `find` on `system.profile`
    - For MongoDB 8.0+: Additional `directShardOperations` role required for sharded clusters

    **Mongolog requirements**

    - MongoDB configured to log slow operations to a file
    - MongoDB server has write permissions to the log directory and file
    - PMM agent has read permissions to the MongoDB log file on the host (no `system.profile` database privileges required)
    - MongoDB monitoring user needs `getCmdLineOpts` privilege to discover the log file path (included in the built-in `clusterMonitor` role)

## Dashboard layout

The Stored metrics view contains three panels:

- [Filters panel](panels/filters.md): narrow results by database, service, or query type
- [Overview panel](panels/overview.md): see query metrics and trends
- [Details panel](panels/details.md): examine individual query performance

## Data collection

Stored metrics collects data once per minute. When collection delays occur, gaps may appear in the sparkline.

## Monitor PMM Server's internal PostgreSQL

By default, Query Analytics hides queries from PMM Server's internal PostgreSQL database. This keeps the focus on your monitored databases.

Enable this when you need to troubleshoot PMM Server performance, check resource usage, or ensure applications are not accidentally using the default `postgres` database. This is particularly useful in [High Availability (HA) deployments](../../install-pmm/HA.md).

To enable:
{.power-number}

1. Go to **Configuration > Settings > Advanced settings**.
2. Switch on the **QAN for PMM Server** option.
3. Open **Query Analytics** and filter by `pmm-server-postgresql` to view queries.

When enabled, you'll see queries related to PMM's internal operations—inventory, settings, advisor checks, alerts, backups, and authentication. These are usually lightweight, but unusual spikes may indicate performance issues.

!!! warning
    Do not use PMM Server's PostgreSQL database for application workloads. Use dedicated databases for your applications.

## See also

- [Real-time Query Analytics](../qan/QAN-realtime-analytics.md)
- [Filters panel](panels/filters.md)
- [Overview panel](panels/overview.md)
- [Details panel](panels/details.md)
---
title: Overview
slug: real-time-analytics
category:
  uri: rta-api
position: 0
privacy:
  view: public
---

Real-time Analytics (RTA) shows the queries currently running on your MongoDB, MySQL and PostgreSQL databases. For MySQL and PostgreSQL, it also shows lock contention, including which statements are waiting for a lock and which transaction is holding it.

RTA shows what's happening on your database right now, while Query Analytics (QAN) analyzes historical query performance to help you spot trends and performance issues over time.

Use the RTA API to:

- start and stop real-time monitoring sessions for MongoDB, MySQL and PostgreSQL services
- search currently executing queries in active sessions
- find MySQL and PostgreSQL statements that are waiting for a lock, and the transactions holding those locks
- list all active monitoring sessions
- integrate live query monitoring into custom dashboards
- automate session management

**Base URL:** `https://your-pmm-server/v1/realtimeanalytics`

**Authentication:** All endpoints require [Bearer token authentication](ref:authentication#bearer-authentication).

## Supported databases

RTA works with MongoDB, MySQL and PostgreSQL services. Before you start a session, make sure PMM Client on the database host meets the minimum version for your database:

| Database | Minimum PMM Client | Service type |
|----------|--------------------|--------------|
| MongoDB | 3.7.0 | `SERVICE_TYPE_MONGODB_SERVICE` |
| MySQL, Percona Server for MySQL and MariaDB | 3.11.0 | `SERVICE_TYPE_MYSQL_SERVICE` |
| PostgreSQL | 3.11.0 | `SERVICE_TYPE_POSTGRESQL_SERVICE` |

RTA isn't available for PMM Server's own PostgreSQL database (`pmm-server-postgresql`).

### Unsupported service types

If you request a service type that RTA doesn't support, such as Valkey, the API returns HTTP status `400`:

```json
{
  "error": "Service type valkey does not support Real-Time Analytics",
  "code": 3,
  "message": "Service type valkey does not support Real-Time Analytics",
  "details": []
}
```

## Real-time vs. stored metrics

| QAN feature | Real-time Analytics (RTA) | Stored metrics  |
|---------|---------------------------|------------------------|
| **Data type** | Currently executing queries | Historical query performance |
| **Time range** | Live data (updates every 1-5 seconds) | Historical data (configurable retention) |
| **Use case** | Identify active issues now | Analyze trends and patterns |
| **Database support** | [MongoDB, MySQL and PostgreSQL](#supported-databases) | MongoDB, MySQL and PostgreSQL |
| **Data retention** | Ephemeral (not stored) | Persistent (stored for analysis) |

## Available endpoints

- [List RTA-compatible services](ref:list-rta-services): retrieve services that support Real-Time Analytics
- [Search real-time analytics queries](ref:search-rta-queries): retrieve currently executing queries from active sessions
- [Manage real-time analytics sessions](ref:manage-rta-sessions): start, stop, and list real-time monitoring sessions

## Common use cases

### Monitor active database operations

When your database is experiencing performance issues, use RTA to see exactly what queries are running and identify bottlenecks in real-time:

1. List available services with `GET /v1/realtimeanalytics/services`
2. Start a session with `POST /v1/realtimeanalytics/sessions:start`
3. Search active queries with `POST /v1/realtimeanalytics/queries:search`
4. Filter results by service to focus on specific instances

### Automated session management

Integrate RTA into your monitoring workflows to automatically enable real-time monitoring during peak hours or when alerts trigger:

1. List existing sessions with `GET /v1/realtimeanalytics/sessions`
2. Start monitoring when needed with `POST /v1/realtimeanalytics/sessions:start`
3. Stop monitoring when done with `POST /v1/realtimeanalytics/sessions:stop`

### Cluster-wide monitoring

Get comprehensive visibility across your entire MongoDB cluster by monitoring all replica set members simultaneously:

1. Start sessions for each service in the cluster
2. Use the cluster filter in list sessions to view cluster status
3. Search queries across all cluster services

### Diagnose lock contention on MySQL

When a MySQL table seems frozen, one transaction is often holding a lock while other statements wait behind it. To find and end that transaction:

1. Start a session for the MySQL service with `POST /v1/realtimeanalytics/sessions:start`.
2. Get the running queries with `POST /v1/realtimeanalytics/queries:search`.
3. In the response, find the queries where `my_sql_payload.blocked_status` is `BLOCKED_STATUS_BLOCKED`. The API doesn't filter by this field, so check it in your own code, for example with `jq`.
4. In each blocked query's `my_sql_payload.blocked_by` list, check the `root` field:

    - If an entry has `root` set to `true`, that transaction isn't waiting for a lock itself, so it's at the head of the chain. More than one entry can have `root` set to `true`.
    - If no entry has `root` set to `true`, each blocking transaction is itself waiting for a lock. Find that transaction's own query in the response, using `blocking_conn_id`, and check its `blocked_by` list the same way until you reach a root. If you come back to the statement you started from, the transactions are deadlocked, and MySQL usually rolls back one of them on its own.

5. End each root transaction so the waiting statements can continue.

### Diagnose lock contention (PostgreSQL)

The same steps work for a PostgreSQL service, using `postgresql_payload` instead of `my_sql_payload`. The `blocking_conn_id` of the root entry is the process ID to pass to `pg_terminate_backend()`, or `0` for a prepared transaction, which you end with `COMMIT PREPARED` or `ROLLBACK PREPARED`.

## Authentication

All RTA endpoints require authentication using service account tokens. Include your token in the request header:

```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/sessions:start" \
  -H "Authorization: Bearer YOUR_SERVICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"service_id": "your-service-id"}'
```

For details about creating and managing service account tokens, see [Authentication with service accounts](https://docs.percona.com/percona-monitoring-and-management/3/api/authentication.html).

## Best practices

### Session management and resource considerations

Each running session adds load to the monitored database and to PMM Server. To keep the impact low:

- start sessions only when actively troubleshooting or for services that need active monitoring
- stop sessions when monitoring is no longer needed or during maintenance windows
- use list sessions to track which services are being monitored
- monitor session status regularly to detect errors or failures
- monitor PMM Server resource usage when multiple sessions are active

### Query search optimization

To minimize server load and improve response times when searching for active queries, follow these guidelines.

- Use `service_ids` filter to limit results to specific services
- Use `limit` parameter to control result set size
- Match your polling interval to the session's collection interval (check `collect_interval` in the session response). Polling slower than the collection interval means you'll miss queries that start and finish between your requests.

## Session status values

Each RTA session has a status that indicates whether it's actively collecting data, has encountered an error, or has been stopped:

| Status | Description |
|--------|-------------|
| `SESSION_STATUS_RUNNING` | Session is actively collecting data |
| `SESSION_STATUS_ERROR` | Session encountered an error |
| `SESSION_STATUS_DOWN` | Session has been stopped or disabled |

## Related resources

- [Real-time Analytics for MongoDB](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics.html)
- [Real-time Analytics for MySQL](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics-mysql.html)
- [Complete PMM API documentation](https://percona-pmm.readme.io/reference/introduction)
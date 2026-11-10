---
title: Search queries
slug: search-rta-queries
content:
  excerpt: Retrieve currently executing queries from active Real-time Analytics sessions.
category:
  uri: rta-api
---

To call this endpoint, include a [service account token](ref:authentication#bearer-authentication) in the `Authorization` header.

## Search queries

`POST /v1/realtimeanalytics/queries:search`

Returns currently executing queries from active Real-time Analytics sessions. This endpoint provides live visibility into database operations happening right now.

### Request body
```json
{
  "service_ids": ["7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"],
  "limit": "100"
}
```

### Parameters

At least one `service_id` must be specified. The `limit` parameter is optional.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_ids` | array of strings | Yes | Filter results to specific service identifiers (at least one required) |
| `limit` | string (int64) | No | Maximum number of queries to return |

### Response

The response lists the queries running in your active sessions. Each query has the same general fields, plus one payload for its database type: `mongo_db_payload` for MongoDB, or `my_sql_payload` for MySQL, Percona Server for MySQL and MariaDB.

Select your database type to see an example. The MySQL example shows a statement waiting for an InnoDB row lock:

```json MongoDB
{
  "queries": [
    {
      "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "service_name": "mongodb-production-rs0",
      "query_id": "conn123:opid456",
      "query_text": "{\"find\": \"transactions\", \"filter\": {\"status\": \"pending\"}}",
      "query_raw_json": "{...}",
      "query_execution_duration": "2.547s",
      "query_collect_time": "2024-03-06T15:23:45Z",
      "client_address": "10.0.1.45:52341",
      "mongo_db_payload": {
        "db_instance_address": "mongodb-rs0:27017",
        "client_app_name": "order-processor",
        "database_name": "orders",
        "collection": "transactions",
        "operation": "find",
        "operation_start_time": "2024-03-06T15:23:42Z",
        "username": "app_user",
        "plan_summary": "IXSCAN { status: 1, created_at: 1 }"
      }
    }
  ]
}
```
```json MySQL
{
  "queries": [
    {
      "service_id": "89008765-c771-44a9-a9a5-e1c5ff51fc36",
      "service_name": "mariadb-production-01",
      "query_id": "8205",
      "query_text": "UPDATE accounts SET balance=999 WHERE id=1",
      "query_raw_json": "{...}",
      "query_execution_duration": "10.420736589s",
      "query_collect_time": "2026-09-19T09:31:27.090750756Z",
      "client_address": "",
      "my_sql_payload": {
        "db_instance_address": "mariadb-01.example.com:3306",
        "program_name": "mysql",
        "database_name": "shop",
        "command": "Query",
        "state": "Updating",
        "username": "app_user@10.0.1.45",
        "rows_examined": "0",
        "rows_sent": "0",
        "full_scan": false,
        "lock_time": "0.000004s",
        "query_text_truncated": false,
        "blocked_status": "BLOCKED_STATUS_BLOCKED",
        "blocked_by": [
          {
            "blocking_conn_id": "8201",
            "blocking_query": "SELECT id, balance FROM accounts WHERE id=1 FOR UPDATE",
            "blocking_command": "Sleep",
            "blocking_username": "batch_user@10.0.1.12",
            "wait_duration": "10.416670s",
            "blocker_transaction_duration": "12.508552340s",
            "root": true,
            "blocking_lock_mode": "X"
          }
        ],
        "locked_table": "shop.accounts",
        "locked_index": "PRIMARY",
        "lock_type": "LOCK_TYPE_ROW",
        "requested_lock_mode": "X"
      }
    }
  ]
}
```

#### Metadata lock example

A DDL statement that waits for an open transaction, such as an `ALTER TABLE`, reports a metadata lock. Compared with a row lock:

- `wait_duration` is `null`, because MySQL doesn't record when a metadata lock wait started.
- `locked_index` is empty, because the lock covers the whole table.
- `blocked_by` lists every transaction ahead in the queue. `root` is `true` for the one that isn't waiting for a lock itself.

```json
{
  "blocked_status": "BLOCKED_STATUS_BLOCKED",
  "blocked_by": [
    {
      "blocking_conn_id": "8201",
      "blocking_query": "SELECT id, balance FROM accounts WHERE id=1 FOR UPDATE",
      "blocking_command": "Sleep",
      "blocking_username": "batch_user@10.0.1.12",
      "wait_duration": null,
      "blocker_transaction_duration": "12.508552340s",
      "root": true,
      "blocking_lock_mode": "SHARED_WRITE"
    },
    {
      "blocking_conn_id": "8205",
      "blocking_query": "UPDATE accounts SET balance=999 WHERE id=1",
      "blocking_command": "Query",
      "blocking_username": "app_user@10.0.1.45",
      "wait_duration": null,
      "blocker_transaction_duration": "10.424386463s",
      "root": false,
      "blocking_lock_mode": "SHARED_WRITE"
    }
  ],
  "locked_table": "shop.accounts",
  "locked_index": "",
  "lock_type": "LOCK_TYPE_METADATA",
  "requested_lock_mode": "EXCLUSIVE"
}
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `queries` | array | List of currently executing queries |
| `queries[].service_id` | string | PMM service identifier |
| `queries[].service_name` | string | PMM service name |
| `queries[].query_id` | string | Unique identifier for the query |
| `queries[].query_text` | string | The text of the query |
| `queries[].query_raw_json` | string | Raw JSON representation of the query |
| `queries[].query_execution_duration` | string | Current query execution time |
| `queries[].query_collect_time` | string (date-time) | When the query data was collected |
| `queries[].client_address` | string | Client address (host:port) |
| `queries[].mongo_db_payload` | object | MongoDB-specific query information |
| `queries[].mongo_db_payload.db_instance_address` | string | MongoDB instance address (host:port) |
| `queries[].mongo_db_payload.client_app_name` | string | Client application name |
| `queries[].mongo_db_payload.database_name` | string | Database name |
| `queries[].mongo_db_payload.collection` | string | Collection name |
| `queries[].mongo_db_payload.operation` | string | Query operation (find, aggregate, update, etc.) |
| `queries[].mongo_db_payload.operation_start_time` | string (date-time) | When the operation started |
| `queries[].mongo_db_payload.username` | string | MongoDB username |
| `queries[].mongo_db_payload.plan_summary` | string | Query execution plan (COLLSCAN vs IXSCAN) |
| `queries[].my_sql_payload` | object | MySQL-specific query information |
| `queries[].my_sql_payload.db_instance_address` | string | MySQL instance address, as `host:port` |
| `queries[].my_sql_payload.program_name` | string | Name of the client program |
| `queries[].my_sql_payload.database_name` | string | Database the connection is using |
| `queries[].my_sql_payload.command` | string | Command the connection is running, such as `Query` or `Execute` |
| `queries[].my_sql_payload.state` | string | State of the connection, as shown in the process list |
| `queries[].my_sql_payload.username` | string | MySQL user, as `user@host` |
| `queries[].my_sql_payload.rows_examined` | string (int64) | Rows the statement has read so far. Missing when the server didn't measure it |
| `queries[].my_sql_payload.rows_sent` | string (int64) | Rows the statement has sent to the client so far. Missing when the server didn't measure it |
| `queries[].my_sql_payload.full_scan` | boolean | Whether the statement scanned a table without a usable index. Missing when the server didn't measure it |
| `queries[].my_sql_payload.lock_time` | string | Time the statement has spent waiting for table locks, from `LOCK_TIME` in `events_statements_current`. `null` when the server didn't measure it |
| `queries[].my_sql_payload.query_text_truncated` | boolean | Whether `query_text` might be cut short. MySQL keeps 1024 bytes of a running statement in the process list, or up to `performance_schema_max_sql_text_length` bytes when the `events_statements_current` consumer is on. PMM sends at most 64 KiB. A statement that's exactly as long as the limit is also reported as truncated |
| `queries[].my_sql_payload.blocked_status` | string | Whether the statement is waiting for a lock: `BLOCKED_STATUS_BLOCKED`, `BLOCKED_STATUS_NOT_BLOCKED`, `BLOCKED_STATUS_UNSPECIFIED` or `BLOCKED_STATUS_UNATTRIBUTED`. [Read what each value means](#read-my_sql_payload-fields) |
| `queries[].my_sql_payload.lock_type` | string | Kind of lock the statement is waiting for: `LOCK_TYPE_ROW` or `LOCK_TYPE_METADATA` |
| `queries[].my_sql_payload.locked_table` | string | Table the statement is waiting to lock, as `schema.table` |
| `queries[].my_sql_payload.locked_index` | string | Index the statement is waiting to lock. Empty for table-level and metadata locks |
| `queries[].my_sql_payload.requested_lock_mode` | string | Lock mode the statement is asking for |
| `queries[].my_sql_payload.blocked_by` | array | Transactions that are blocking the statement. Empty unless `blocked_status` is `BLOCKED_STATUS_BLOCKED` |
| `queries[].my_sql_payload.blocked_by[].blocking_conn_id` | string (int64) | Connection ID of the blocking connection |
| `queries[].my_sql_payload.blocked_by[].blocking_query` | string | Statement the blocking connection is running, or the last one it ran |
| `queries[].my_sql_payload.blocked_by[].blocking_query_truncated` | boolean | Whether `blocking_query` might be cut short, by the same limits as `query_text_truncated` |
| `queries[].my_sql_payload.blocked_by[].blocking_command` | string | Command the blocking connection is running. `Sleep` means it's idle inside an open transaction |
| `queries[].my_sql_payload.blocked_by[].blocking_username` | string | MySQL user of the blocking connection |
| `queries[].my_sql_payload.blocked_by[].blocking_lock_mode` | string | Lock mode the blocking transaction holds |
| `queries[].my_sql_payload.blocked_by[].wait_duration` | string | How long the statement has been waiting for this transaction. `null` for metadata locks |
| `queries[].my_sql_payload.blocked_by[].blocker_transaction_duration` | string | How long the blocking transaction has been open |
| `queries[].my_sql_payload.blocked_by[].root` | boolean | Whether the blocking transaction isn't waiting for a lock itself, which puts it at the head of the chain |

#### Read `my_sql_payload` fields

If you search queries on MySQL, Percona Server for MySQL or MariaDB services, some `my_sql_payload` fields can be missing or report an unknown value, and some differ depending on the server. 

Check these fields before you act on the data:

- `rows_examined`, `rows_sent` or `full_scan` is missing, or `lock_time` is `null`

    The server didn't measure the value. Treat it as unknown, not as zero rows or no full scan. A server that measured these fields reports a real `0` or `false`. Whether they're measured depends on the `events_statements_current` consumer:

    - MySQL and Percona Server for MySQL: the consumer is on by default.
    - MariaDB: the consumer is off by default, so these fields are missing. To collect them, [enable the missing Performance Schema settings](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics-mysql.html#get-complete-data-on-mysql-57-and-mariadb).

- `blocked_status` is `BLOCKED_STATUS_UNSPECIFIED`

    RTA couldn't read one of its two lock sources, so it doesn't know whether the statement is waiting. Treat it as unknown, not as not blocked. This can happen when the PMM database user is missing a privilege, when the server doesn't support a lock source, or when the lock information is too large to read. The most common cause is that the `wait/lock/metadata/sql/mdl` instrument is off:

    - MySQL and Percona Server for MySQL 8.0 and later: the instrument is on by default.
    - MySQL and Percona Server for MySQL 5.7, and MariaDB: the instrument is off by default. To turn it on, [enable the missing Performance Schema settings](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics-mysql.html#get-complete-data-on-mysql-57-and-mariadb).

- `blocked_status` is `BLOCKED_STATUS_UNATTRIBUTED`

    RTA read both lock sources, but the connection moved on to a later statement between the two reads, so RTA can't tell whether the reported statement was waiting. Treat it as unknown for this refresh. The next collection reads both again, and you don't need to change any settings.

- Lock modes in `blocking_lock_mode` and `requested_lock_mode`

    The lock mode format depends on the server:

    - MySQL and Percona Server for MySQL: tell a record lock apart from a gap lock, for example `X,REC_NOT_GAP` or `X,GAP`.
    - MariaDB: reports only `X` or `S`.

### Examples

#### Get queries for a specific service
```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/queries:search" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_ids": ["7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"]
  }'
```

#### Find blocked statements on a MySQL service

Search the service, then keep the queries whose `my_sql_payload.blocked_status` is `BLOCKED_STATUS_BLOCKED`. The transaction to resolve is the `blocked_by` entry with `"root": true`:

```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/queries:search" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_ids": ["89008765-c771-44a9-a9a5-e1c5ff51fc36"]
  }' \
  | jq '.queries[]
        | select(.my_sql_payload.blocked_status == "BLOCKED_STATUS_BLOCKED")
        | {query_text,
           locked_table: .my_sql_payload.locked_table,
           lock_type: .my_sql_payload.lock_type,
           root_blockers: [.my_sql_payload.blocked_by[] | select(.root) | .blocking_conn_id]}'
```

#### Filter by multiple services
```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/queries:search" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_ids": [
      "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "8b4f0d55-9fce-5e4g-0f32-6d9e8c2b3f5g"
    ]
  }'
```

#### Limit number of results
```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/queries:search" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_ids": ["7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"],
    "limit": "50"
  }'
```

### Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | Query data retrieved successfully |
| `400` | Bad Request | Invalid request parameters |
| `401` | Unauthorized | Missing or invalid authentication token |
| `403` | Forbidden | Insufficient permissions to access RTA data |
| `500` | Internal Server Error | Server error processing request |

#### Error response format
```json
{
  "code": 3,
  "message": "Invalid limit value",
  "details": []
}
```

### Troubleshooting

#### No queries returned

The API returns an empty result set even though you expect to see query data. This can happen when no active Real-time Analytics sessions are running, no queries are currently executing, or the service IDs don't match any active sessions.

**Solutions:**

1. Verify sessions are running with `GET /v1/realtimeanalytics/sessions`
2. Check the service supports RTA with `GET /v1/realtimeanalytics/services`
3. Start a session with `POST /v1/realtimeanalytics/sessions:start`
4. Check that services have active database traffic

#### Empty mongo_db_payload

Query data is returned but the MongoDB-specific payload is empty. This typically occurs when query data is not yet available from the collection cycle.

**Solutions:**

1. Wait for the next data collection cycle
2. Check PMM agent logs for collection errors
3. Verify the MongoDB user has the required permissions for `$currentOp`. See [MongoDB currentOp Access Control](https://www.mongodb.com/docs/manual/reference/operator/aggregation/currentOp/#access-control) for details.

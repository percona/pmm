---
title: Search queries
slug: search-rta-queries
content:
  excerpt: Retrieve currently executing queries from active Real-time Analytics sessions.
category:
  uri: rta-api
---

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
```json
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

#### MySQL response

Queries from a MySQL service carry `my_sql_payload` instead. This example shows a statement blocked on an InnoDB row lock:

```json
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

A statement queued behind a DDL reports a metadata lock instead. Metadata-lock waits carry no `wait_duration`, because the server records no timestamp for them, and no `locked_index`, because the lock is taken on the table as a whole. `blocked_by` lists every transaction ahead in the queue, with `root` marking the one that is not itself waiting:

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

> 🚧 Important
> 
> `rows_examined`, `rows_sent` and `full_scan` are omitted from the payload when the server did not measure them, which is the default on MariaDB because the `events_statements_current` consumer ships disabled. Absent means unknown: treat it as "not measured" rather than as zero rows or no full scan. A server that did measure reports a real `0` or `false`, so the two are distinguishable.
> 
> `blocked_status` is `BLOCKED_STATUS_UNSPECIFIED` when RTA could not read one of its two lock sources, and therefore cannot say whether the statement is waiting. Treat it as "unknown", not as "not blocked". The most common cause is the `wait/lock/metadata/sql/mdl` instrument being disabled, which is the default on MariaDB.
>
> `blocked_status` is `BLOCKED_STATUS_UNATTRIBUTED` when the lock sources were read, but the connection was waiting for a lock on behalf of a later statement than the one reported: it moved on between the statement read and the lock read. Treat it as "unknown for this refresh"; the next collection reads both again. No configuration change is needed.
> 
> On MySQL and Percona Server, lock modes distinguish a record lock from a gap lock (`X,REC_NOT_GAP`, `X,GAP`). MariaDB reports plain `X` or `S`.

#### PostgreSQL response

Queries from a PostgreSQL service carry `postgresql_payload`, read from `pg_stat_activity`. `query_id` is the session's process ID. This example shows an `UPDATE` waiting for a row held by a session that is idle in an open transaction:

```json
{
  "queries": [
    {
      "service_id": "39c8e124-284a-4871-b9c0-32d6708dee4a",
      "service_name": "postgresql-production-01",
      "query_id": "7750",
      "query_text": "UPDATE accounts SET balance = 999 WHERE id = 1",
      "query_raw_json": "{...}",
      "query_execution_duration": "40.685233s",
      "query_collect_time": "2026-10-10T06:47:12.101928Z",
      "client_address": "10.0.1.45:51234",
      "postgresql_payload": {
        "database_name": "shop",
        "username": "app_user",
        "application_name": "psql",
        "state": "active",
        "wait_event_type": "Lock",
        "wait_event": "transactionid",
        "pid": 7750,
        "query_id": "-3248750928237446514",
        "transaction_start_time": "2026-10-10T06:46:31.416695Z",
        "query_start_time": "2026-10-10T06:46:31.416695Z",
        "query_text_truncated": false,
        "blocked_status": "BLOCKED_STATUS_BLOCKED",
        "blocked_by": [
          {
            "blocking_conn_id": "7739",
            "blocking_query": "UPDATE accounts SET balance = 0 WHERE id = 1",
            "blocking_command": "idle in transaction",
            "blocking_username": "batch_user",
            "wait_duration": "40.685233s",
            "blocker_transaction_duration": "52.508552s",
            "root": true
          }
        ],
        "db_instance_address": "pg-01.example.com:5432"
      }
    }
  ]
}
```

> 📘 Info
>
> `blocked_by` holds the sessions that `pg_blocking_pids()` reports for this one and, transitively, theirs, with `root` marking the ones that are not waiting themselves. `blocking_conn_id` is the blocker's process ID, or `0` for a prepared transaction. `blocking_command` is the blocker's state, or its backend type for a background process such as an autovacuum worker. PostgreSQL reports no lock facts, so `lock_type`, `locked_table` and the lock modes are MySQL-only.
>
> RTA lists at most 1000 sessions per collection, longest-running first, and at most 5000 `blocked_by` entries. Every blocked session gets its roots first. Past that limit a session lists only its roots, and if even those do not fit, its `blocked_status` is `BLOCKED_STATUS_UNSPECIFIED`. `BLOCKED_STATUS_UNATTRIBUTED` is never reported for PostgreSQL.

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
| `queries[].my_sql_payload.db_instance_address` | string | MySQL instance address (host:port) |
| `queries[].my_sql_payload.program_name` | string | Client program name |
| `queries[].my_sql_payload.database_name` | string | Database name |
| `queries[].my_sql_payload.command` | string | Process list command (Query, Execute, etc.) |
| `queries[].my_sql_payload.state` | string | Process list state |
| `queries[].my_sql_payload.username` | string | MySQL user (`user@host`) |
| `queries[].my_sql_payload.rows_examined` | string (int64) | Rows read to produce the result. Absent when the server did not measure it |
| `queries[].my_sql_payload.rows_sent` | string (int64) | Rows returned to the client. Absent when the server did not measure it |
| `queries[].my_sql_payload.full_scan` | boolean | Whether the statement scanned without a usable index. Absent when the server did not measure it |
| `queries[].my_sql_payload.lock_time` | string | Time the statement has waited for table locks (`LOCK_TIME` from `events_statements_current`). Absent when the server did not measure it |
| `queries[].my_sql_payload.query_text_truncated` | boolean | `query_text` is not the whole statement. MySQL keeps 1024 bytes of a running statement in the process list, and up to `performance_schema_max_sql_text_length` bytes while the `events_statements_current` consumer is enabled; PMM sends at most 64 KiB. A statement exactly as long as the limit is reported as truncated too |
| `queries[].my_sql_payload.blocked_status` | string | `BLOCKED_STATUS_BLOCKED`, `BLOCKED_STATUS_NOT_BLOCKED`, `BLOCKED_STATUS_UNSPECIFIED` when a lock source could not be read, or `BLOCKED_STATUS_UNATTRIBUTED` when the wait found belonged to a later statement on the same connection |
| `queries[].my_sql_payload.lock_type` | string | `LOCK_TYPE_ROW` or `LOCK_TYPE_METADATA` |
| `queries[].my_sql_payload.locked_table` | string | Contended table (`schema.table`) |
| `queries[].my_sql_payload.locked_index` | string | Contended index; empty for table- and metadata-level locks |
| `queries[].my_sql_payload.requested_lock_mode` | string | Lock mode this statement is requesting |
| `queries[].my_sql_payload.blocked_by` | array | Transactions blocking this statement |
| `queries[].my_sql_payload.blocked_by[].blocking_conn_id` | string (int64) | Connection ID of the blocker |
| `queries[].my_sql_payload.blocked_by[].blocking_query` | string | Statement the blocker is running, or its last statement |
| `queries[].my_sql_payload.blocked_by[].blocking_query_truncated` | boolean | `blocking_query` is not the whole statement; see `query_text_truncated` |
| `queries[].my_sql_payload.blocked_by[].blocking_command` | string | Blocker's process list command |
| `queries[].my_sql_payload.blocked_by[].blocking_username` | string | Blocker's MySQL user |
| `queries[].my_sql_payload.blocked_by[].blocking_lock_mode` | string | Lock mode the blocker holds |
| `queries[].my_sql_payload.blocked_by[].wait_duration` | string | How long this statement has been waiting |
| `queries[].my_sql_payload.blocked_by[].blocker_transaction_duration` | string | How long the blocking transaction has been open |
| `queries[].my_sql_payload.blocked_by[].root` | boolean | Blocker is not itself waiting — the head of the chain |
| `queries[].postgresql_payload` | object | PostgreSQL-specific query information, read from `pg_stat_activity` |
| `queries[].postgresql_payload.db_instance_address` | string | PostgreSQL instance PMM connects to: host:port, or the socket directory |
| `queries[].postgresql_payload.database_name` | string | Database the session is connected to |
| `queries[].postgresql_payload.username` | string | PostgreSQL user of the session |
| `queries[].postgresql_payload.application_name` | string | Client `application_name` |
| `queries[].postgresql_payload.state` | string | Session state: `active`, `idle in transaction`, `idle in transaction (aborted)` or `fastpath function call` |
| `queries[].postgresql_payload.wait_event_type` | string | What the session is waiting for, for example `Lock`, `IO` or `Client`; empty when not waiting |
| `queries[].postgresql_payload.wait_event` | string | The wait event within `wait_event_type` |
| `queries[].postgresql_payload.pid` | integer | Process ID of the session's backend |
| `queries[].postgresql_payload.query_id` | string | `pg_stat_activity.query_id`; empty before PostgreSQL 14, or when the server does not compute query IDs (`compute_query_id`) |
| `queries[].postgresql_payload.transaction_start_time` | string (date-time) | When the current transaction started |
| `queries[].postgresql_payload.query_start_time` | string (date-time) | When the current or last statement started. For a session idle in transaction, `query_execution_duration` is the transaction's age |
| `queries[].postgresql_payload.query_text_truncated` | boolean | `query_text` is cut at `track_activity_query_size` bytes (1024 by default) |
| `queries[].postgresql_payload.blocked_status` | string | `BLOCKED_STATUS_BLOCKED` while waiting for a heavyweight lock, otherwise `BLOCKED_STATUS_NOT_BLOCKED`; `BLOCKED_STATUS_UNSPECIFIED` when the session's blockers did not fit the limit |
| `queries[].postgresql_payload.blocked_by` | array | Sessions holding up this one, ordered by process ID; same fields as `my_sql_payload.blocked_by[]` except `blocking_lock_mode` |

> 📘 Info
> 
> Each query carries exactly one payload, matching the service type: `mongo_db_payload` for MongoDB services, `my_sql_payload` for MySQL, Percona Server and MariaDB services, and `postgresql_payload` for PostgreSQL services.

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
           root_blocker: (.my_sql_payload.blocked_by[] | select(.root) | .blocking_conn_id)}'
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

To get the authentication token, check [Authentication](ref:authentication).
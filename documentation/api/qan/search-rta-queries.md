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
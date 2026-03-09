---
title: Get real-time query data
slug: getrealtimequerydata
content:
  excerpt: Retrieve currently executing queries for monitored MongoDB services.
category:
  uri: rta-api
---

## Get real-time query data

`GET /v1/realtime/query-data`

Retrieves a list of currently executing queries on your monitored MongoDB services. This endpoint provides real-time visibility into active database operations.

### Query parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster` | string | No | Filter results to a specific MongoDB cluster |
| `service` | string | No | Filter results to a specific MongoDB service |

**Note:** Both `cluster` and `service` are optional. If neither is specified, the endpoint returns data for all monitored services with RTA enabled.

### Response
```json
{
  "queries": [
    {
      "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "service_name": "mongodb-production-rs0",
      "cluster": "production-cluster",
      "database": "orders",
      "collection": "transactions",
      "operation": "find",
      "query": "{\"status\": \"pending\", \"created_at\": {\"$gte\": ISODate(\"2024-03-06T00:00:00Z\")}}",
      "duration_ms": 2547,
      "namespace": "orders.transactions",
      "client": "10.0.1.45:52341",
      "app_name": "order-processor",
      "plan_summary": "IXSCAN { status: 1, created_at: 1 }",
      "num_yields": 12,
      "locks_acquired": {
        "Global": "r",
        "Database": "r",
        "Collection": "r"
      }
    },
    {
      "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "service_name": "mongodb-production-rs0",
      "cluster": "production-cluster",
      "database": "users",
      "collection": "sessions",
      "operation": "update",
      "query": "{\"session_id\": \"a7f3e9c4-12ab-4d3f-9e21-5c8d7b1a2e4f\"}",
      "duration_ms": 145,
      "namespace": "users.sessions",
      "client": "10.0.1.67:41223",
      "app_name": "web-api",
      "plan_summary": "IXSCAN { session_id: 1 }",
      "num_yields": 0,
      "locks_acquired": {
        "Global": "w",
        "Database": "w",
        "Collection": "w"
      }
    }
  ],
  "timestamp": "2024-03-06T15:23:45Z",
  "total_queries": 2
}
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `queries` | array | List of currently executing queries |
| `queries[].service_id` | string | Unique identifier for the MongoDB service |
| `queries[].service_name` | string | Human-readable service name |
| `queries[].cluster` | string | Cluster name (if service is part of a cluster) |
| `queries[].database` | string | Database where the query is executing |
| `queries[].collection` | string | Collection being queried |
| `queries[].operation` | string | Operation type: `find`, `update`, `insert`, `delete`, `aggregate`, `command` |
| `queries[].query` | string | Query filter or command document |
| `queries[].duration_ms` | integer | Time query has been running (milliseconds) |
| `queries[].namespace` | string | Full namespace (database.collection) |
| `queries[].client` | string | Client connection address and port |
| `queries[].app_name` | string | Application name from connection string |
| `queries[].plan_summary` | string | Query execution plan summary |
| `queries[].num_yields` | integer | Number of times query yielded control |
| `queries[].locks_acquired` | object | Lock types acquired (r=read, w=write) |
| `timestamp` | string | Time when data was collected (ISO 8601 format) |
| `total_queries` | integer | Total number of active queries returned |

### Examples

#### Get all active queries
```bash
curl -X GET "https://your-pmm-server/v1/realtime/query-data" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json"
```

#### Filter by cluster
```bash
curl -X GET "https://your-pmm-server/v1/realtime/query-data?cluster=production-cluster" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json"
```

#### Filter by service
```bash
curl -X GET "https://your-pmm-server/v1/realtime/query-data?service=mongodb-production-rs0" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json"
```

### Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | Query data retrieved successfully |
| `401` | Unauthorized | Missing or invalid authentication token |
| `403` | Forbidden | Insufficient permissions to access RTA data |
| `404` | Not Found | Specified cluster or service not found |
| `500` | Internal Server Error | Server error processing request |

#### Error response format
```json
{
  "code": 5,
  "message": "Service 'mongodb-invalid' not found",
  "details": []
}
```

### Troubleshooting

#### No queries returned

**Possible causes:**
- RTA is not enabled for the service or cluster
- No queries are currently executing
- Service is not actively monitored by PMM

**Solutions:**
1. Verify RTA is enabled using `POST /v1/realtime/change`
2. Check that the service exists in PMM inventory
3. Confirm the service has active database traffic

#### Incomplete query data

**Possible causes:**
- MongoDB profiler not configured correctly
- Insufficient permissions for PMM monitoring user

**Solutions:**
1. Verify MongoDB profiling level is set to 1 or 2
2. Check PMM agent has necessary MongoDB permissions
3. Review PMM agent logs for connection issues

To get the authentication token, check [Authentication](ref:authentication).
---
title: Manage sessions
slug: manage-rta-sessions

content:
  excerpt: Start, stop, and list Real-time Analytics sessions.
category:
  uri: rta-api
---

## Start session

`POST /v1/realtimeanalytics/sessions:start`

Starts a Real-time Analytics (RTA) session for a specified MongoDB or MySQL service. Once started, the session will continuously collect data about currently executing queries.

### Request body
```json
{
  "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_id` | string | Yes | Service identifier to start RTA session for |

### Response
```json
{
  "session": {
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
    "service_name": "mongodb-production-rs0",
    "cluster_name": "production-cluster",
    "start_time": "2024-03-06T15:20:00Z",
    "collect_interval": "2s",
    "status": "SESSION_STATUS_RUNNING",
    "service_type": "SERVICE_TYPE_MONGODB_SERVICE"
  }
}
```

Starting a session for a MySQL service returns the same shape with `service_type` set to `SERVICE_TYPE_MYSQL_SERVICE`:

```json
{
  "session": {
    "service_id": "89008765-c771-44a9-a9a5-e1c5ff51fc36",
    "service_name": "mariadb-production-01",
    "cluster_name": "",
    "start_time": "2026-09-18T19:38:29.566969Z",
    "collect_interval": "2s",
    "status": "SESSION_STATUS_RUNNING",
    "service_type": "SERVICE_TYPE_MYSQL_SERVICE"
  }
}
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `session` | object | The started session details |
| `session.service_id` | string | Service identifier |
| `session.service_name` | string | Service name |
| `session.cluster_name` | string | Cluster name the service belongs to |
| `session.start_time` | string (date-time) | When the session started |
| `session.collect_interval` | string | Query collection interval |
| `session.status` | string | Session status (see status values below) |
| `session.service_type` | string | `SERVICE_TYPE_MONGODB_SERVICE` or `SERVICE_TYPE_MYSQL_SERVICE` |

### Example
```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/sessions:start" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"
  }'
```

---

## Stop session

`POST /v1/realtimeanalytics/sessions:stop`

Stops a RTA session for a specified service.

### Request body
```json
{
  "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `service_id` | string | Yes | Service identifier to stop RTA session for |

### Response
```json
{}
```

**Success response:** Empty object with HTTP 200 status code

### Example
```bash
curl -X POST "https://your-pmm-server/v1/realtimeanalytics/sessions:stop" \
  -H "Authorization: Bearer glsa_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f"
  }'
```

---

## List sessions

`GET /v1/realtimeanalytics/sessions`

Returns the list of all currently running Real-time Analytics sessions with their details including service, cluster, and status information.

### Query parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | No | Filter sessions by cluster name |

### Response
```json
{
  "sessions": [
    {
      "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
      "service_name": "mongodb-production-rs0",
      "cluster_name": "production-cluster",
      "start_time": "2024-03-06T15:20:00Z",
      "collect_interval": "2s",
      "status": "SESSION_STATUS_RUNNING",
      "service_type": "SERVICE_TYPE_MONGODB_SERVICE"
    },
    {
      "service_id": "89008765-c771-44a9-a9a5-e1c5ff51fc36",
      "service_name": "mariadb-production-01",
      "cluster_name": "",
      "start_time": "2026-09-18T19:38:29.566969Z",
      "collect_interval": "2s",
      "status": "SESSION_STATUS_RUNNING",
      "service_type": "SERVICE_TYPE_MYSQL_SERVICE"
    }
  ]
}
```

### Response schema

| Field | Type | Description |
|-------|------|-------------|
| `sessions` | array | List of active sessions |
| `sessions[].service_id` | string | Service identifier |
| `sessions[].service_name` | string | Service name |
| `sessions[].cluster_name` | string | Cluster name |
| `sessions[].start_time` | string (date-time) | When the session started |
| `sessions[].collect_interval` | string | Query collection interval |
| `sessions[].status` | string | Session status |
| `sessions[].service_type` | string | `SERVICE_TYPE_MONGODB_SERVICE` or `SERVICE_TYPE_MYSQL_SERVICE` |

### Examples

#### List all sessions
```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/sessions" \
  -H "Authorization: Bearer glsa_xxxxx"
```

#### Filter by cluster
```bash
curl -X GET "https://your-pmm-server/v1/realtimeanalytics/sessions?cluster_name=production-cluster" \
  -H "Authorization: Bearer glsa_xxxxx"
```

## Session status values

| Status | Description |
|--------|-------------|
| `SESSION_STATUS_RUNNING` | Session is actively collecting query data |
| `SESSION_STATUS_ERROR` | Session encountered an error |
| `SESSION_STATUS_DOWN` | Session has been stopped or disabled |
| `SESSION_STATUS_UNSPECIFIED` | Status is unknown |

## Error responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| `200` | Success | Operation completed successfully |
| `400` | Bad Request | Invalid request (missing service_id, etc.) |
| `401` | Unauthorized | Missing or invalid authentication token |
| `403` | Forbidden | Insufficient permissions |
| `404` | Not Found | Service not found |
| `500` | Internal Server Error | Server error processing request |

### Error response format
```json
{
  "code": 5,
  "message": "Service with ID '...' not found",
  "details": []
}
```

## Troubleshooting

### Session won't start

You're unable to start an RTA session. This typically happens when the service doesn't exist in PMM inventory, the PMM Client version is too old (**< 3.7.0** for MongoDB, **< 3.9.0** for MySQL), or the exporter is not configured.

For MySQL services, a session also fails to start when `performance_schema` is disabled on the monitored server, or when the PMM monitoring user lacks the `SELECT` and `PROCESS` privileges. The session error message names the check that failed.

**Solutions:**

1. List available services using `GET /v1/realtimeanalytics/services` to verify the service supports RTA
2. Check PMM Client version: `pmm-admin status`
3. Verify MongoDB exporter is running in PMM Inventory

### Session shows ERROR status

A session was started successfully but now shows ERROR status. This usually indicates that the PMM agent connection was lost or the database user has insufficient permissions.

**Solutions:**

1. Verify PMM agent status: `pmm-admin status`
2. Check PMM agent logs for errors
3. Verify network connectivity between PMM agent and MongoDB
4. Confirm MongoDB user has the required permissions for `$currentOp`. See [MongoDB currentOp Access Control](https://www.mongodb.com/docs/manual/reference/operator/aggregation/currentOp/#access-control) for details.
5. For MySQL services, confirm the monitoring user has `SELECT` and `PROCESS`, and that `performance_schema` is enabled. See [Real-time Analytics for MySQL](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics-mysql.html).

To get the authentication token, check [Authentication](ref:authentication).
---
title: Manage sessions
slug: manage-rta-sessions

content:
  excerpt: Start, stop, and list Real-time Analytics (RTA) sessions.
category:
  uri: rta-api
---

To call these endpoints, include a [service account token](ref:authentication#bearer-authentication) in the `Authorization` header.

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

Select your database type to see an example response. Both have the same fields, and only the `service_type` value is different:

```json MongoDB
{
  "session": {
    "service_id": "7a3e9c44-12ab-4d3f-9e21-5c8d7b1a2e4f",
    "service_name": "mongodb-production-rs0",
    "cluster_name": "production-cluster",
    "start_time": "2024-03-06T15:20:00Z",
    "collect_interval": "2s",
    "status": "SESSION_STATUS_RUNNING",
    "service_type": "SERVICE_TYPE_MONGODB_SERVICE",
    "status_message": ""
  }
}
```
```json MySQL
{
  "session": {
    "service_id": "89008765-c771-44a9-a9a5-e1c5ff51fc36",
    "service_name": "mariadb-production-01",
    "cluster_name": "",
    "start_time": "2026-09-18T19:38:29.566969Z",
    "collect_interval": "2s",
    "status": "SESSION_STATUS_RUNNING",
    "service_type": "SERVICE_TYPE_MYSQL_SERVICE",
    "status_message": ""
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
| `session.status_message` | string | Explains what went wrong with the session. If `status` is `SESSION_STATUS_ERROR`, it tells you why the session didn't start. If `status` is `SESSION_STATUS_RUNNING`, it tells you which data the session found it can't collect when it started. Problems found later appear only in the pmm-agent log. Empty when there's nothing to report. |

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

Stops an RTA session for a specified service.

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
| `sessions[].status_message` | string | Explains what went wrong with the session, in the same way as `session.status_message` when you [start a session](#start-session). |

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

If PMM can't start a session, the request fails with an HTTP status and an error message. Match the message to its cause and fix it. PMM stops at the first check that fails, so after you fix one cause, you might get a different message:

- HTTP `404`: `Service with ID "<ID>" not found.`

    Check the service ID, or [list the services you can monitor](ref:list-rta-services) to get the correct one.

- HTTP `400`: `Service <ID> of type <type> does not support Real-Time Analytics`

    RTA works only with MongoDB and MySQL services. To find a service you can use, [list the services you can monitor](ref:list-rta-services).

- HTTP `400`: `Service <ID> of type <type> doesn't have agents to retrieve credentials and pmm-agent ID`

    RTA reuses the credentials of the service's exporter or QAN agent, and this service has neither. In PMM, go to **Inventory > Services** and check that the service has an exporter or a QAN agent.

- HTTP `400`: `Service <name> (id <ID>) has pmm-agent with version <version> not supporting Real-Time Analytics; pmm-agent <version> or later is required.`

    PMM Client on the database host is too old. RTA needs PMM Client 3.7.0 or later for MongoDB, and 3.11.0 or later for MySQL. [Upgrade PMM Client](https://docs.percona.com/percona-monitoring-and-management/3/pmm-upgrade/upgrade_client.html) on that host. Adding an RTA agent with `POST /v1/inventory/agents`, using `rta_mysql_agent` or `rta_mongodb_agent`, fails with the same error when that PMM Agent has reported an older version.

### Session shows ERROR status

If a session started but its `status` is now `SESSION_STATUS_ERROR`, [list sessions](#list-sessions) to read its `status_message`. Then match the message to its cause and fix it:

- `Service <name> (id <ID>) has pmm-agent with version <version> not supporting Real-Time Analytics; pmm-agent <version> or later is required.`

    The RTA agent runs on a PMM Agent that's too old. This can happen if you added the RTA agent through the inventory API before that PMM Agent first connected. [Upgrade PMM Client](https://docs.percona.com/percona-monitoring-and-management/3/pmm-upgrade/upgrade_client.html) on the database host.

- `Cannot connect to MySQL: <reason>`

    This message appears only for MySQL services. PMM Agent can't connect to the MySQL server. Check that the MySQL server is running and reachable from the database host, and that the PMM database user's credentials are correct.

- `Real-Time Analytics is not supported for this instance: <reason>`

    This message appears only for MySQL services. PMM can't read `performance_schema` on the MySQL server. For example: `Real-Time Analytics is not supported for this instance: performance_schema is disabled; it is required for Real-Time Analytics`. Check that `performance_schema` is enabled and that the PMM database user has the `SELECT` privilege. If anything is missing, [set up the MySQL requirements for RTA](https://docs.percona.com/percona-monitoring-and-management/3/use/qan/QAN-realtime-analytics-mysql.html#check-the-service-requirements).

- No message, or a different message

    MongoDB sessions don't report a `status_message`, so for MongoDB, start here. This usually means PMM Agent lost its connection, or the database user doesn't have the required permissions. On the database host, check these in order:

    1. Run `pmm-admin status` to check that PMM Agent is connected to PMM Server.
    2. Check the PMM Agent logs for errors.
    3. Check that the host can reach the database.
    4. (MongoDB only) Check that the PMM database user can run `$currentOp`. [Review the required MongoDB permissions](https://www.mongodb.com/docs/manual/reference/operator/aggregation/currentOp/#access-control).

---
title: Check connectivity
slug: sep-connectivity-check
category:
  uri: sep-tasks-api
position: 4
---

Before running a task, verify that an executor host can reach the target database.

```
POST /connectivity-check/
```

Runs a lightweight check on the specified executor host that attempts to connect to the database host and port, then returns immediately with the result.

## Request body

```json
{
  "target": "db-host-01",
  "host": "10.0.0.5",
  "port": 3306,
  "service_type": "mysql",
  "timeout": 30
}
```

| Field | Type | Description |
|-------|------|-------------|
| `target` | string | The executor host to run the check on. |
| `host` | string | The database host address to connect to. |
| `port` | integer | The database port number. |
| `service_type` | string | Database type: `mysql`, `postgresql`, or `mongodb`. |
| `timeout` | integer | How long to wait for the connection, in seconds (1 to 60). Default: 30. |

## Response

```json
{
  "success": true,
  "error": null,
  "task_history_id": 99
}
```

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Whether the connectivity check succeeded. |
| `error` | string | Error message if the check failed. `null` on success. |
| `task_history_id` | integer | ID of the history record for this check. Use it to retrieve logs if you need to investigate a failure. |

## Example

```shell
curl -sk -X POST https://<pmm-server>/sep/connectivity-check/ \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{
       "target": "db-host-01",
       "host": "10.0.0.5",
       "port": 3306,
       "service_type": "mysql",
       "timeout": 30
     }'
```

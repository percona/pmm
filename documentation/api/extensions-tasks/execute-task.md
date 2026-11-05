---
title: Execute a task
slug: extensions-execute-task
category:
  uri: extensions-tasks-api
position: 1
---

Send a task for execution on a target host. Requires the **Admin** role.

```
POST /extensions/api/apps/{app}/{task_name}/execute
```

## Path parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `app` | string | The app that owns the task (for example, `mysql_backups`). |
| `task_name` | string | The name of the task to execute. |

## Request body

```json
{
  "eta": "<future ISO 8601 datetime, e.g. 2030-01-01T12:00:00Z>",
  "chain_task_names": ["string"],
  "chain_on_failure": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `eta` | datetime | Earliest time to start the task. Omit to run immediately. If the time has already passed when the executor picks up the task, it runs immediately instead of waiting. |
| `chain_task_names` | array of strings | Task names to execute sequentially after this one completes, in order. |
| `chain_on_failure` | boolean | If `true`, the chain continues even when a task fails, stops, or is lost. Default: `false` (chain only on success). |

To change which host runs the task, edit the task in the UI before executing. Any `target` field in the request body is ignored.

## Response

Returns HTTP 201 with the dispatched run details.

```json
{
  "task_name": "<task_name>",
  "task_id": 42,
  "status": "pending",
  "created_at": "2026-11-01T02:00:00Z"
}
```

Use the returned `task_id` to [monitor the run](ref:extensions-task-history).

## Example

Replace `<task_name>` with the task name you assigned when creating the task in **Apps > MySQL Backups**.

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/apps/mysql_backups/<task_name>/execute \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{}'
```

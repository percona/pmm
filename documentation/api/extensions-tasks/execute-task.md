---
title: Execute a task
slug: extensions-execute-task
category:
  uri: extensions-tasks-api
position: 1
---

Send a task for execution on a target host.

```
POST /api/apps/{app}/{task_name}/execute
```

## Path parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `app` | string | The app that owns the task (for example, `mysql_backups`). |
| `task_name` | string | The name of the task to execute. |

## Request body

```json
{
  "eta": "2026-09-08T12:00:00Z",
  "chain_task_names": ["string"],
  "chain_on_failure": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `eta` | datetime | Earliest time to execute the task. Omit to execute immediately. |
| `chain_task_names` | array of strings | Task names to execute sequentially after this one completes, in order. |
| `chain_on_failure` | boolean | If `true`, the chain continues even when a task fails, stops, or is lost. Default: `false` (chain only on success). |

## Response

Returns HTTP 201 with the dispatched run details.

```json
{
  "task_name": "mysql-backup-xtrabackup",
  "task_id": 42,
  "status": "pending",
  "created_at": "2026-09-08T12:00:00Z"
}
```

Use the returned `task_id` to [monitor the run](ref:extensions-task-history).

## Example

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/apps/mysql_backups/<task_name>/execute \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{}'
```

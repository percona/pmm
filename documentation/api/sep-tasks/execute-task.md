---
title: Execute a task
slug: sep-execute-task
category:
  uri: sep-tasks-api
position: 1
---

Send a task for execution on a target host.

```
POST /execute/{task_name}
```

## Path parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `task_name` | string | The name of the task to execute. Use [List tasks](ref:sep-list-tasks) to find available task names. |

## Request body

```json
{
  "meta": {},
  "payload": "string",
  "eta": "2026-09-08T12:00:00Z",
  "chain_task_names": ["string"],
  "chain_on_failure": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `meta` | object | Key-value pairs passed to the task at runtime (for example, target host, backup directory, MySQL service ID). The accepted keys depend on the specific task. |
| `payload` | string | Optional HCL, JSON, or YAML job specification to override the task's default payload. |
| `eta` | datetime | Earliest time to execute the task. Omit to execute immediately. |
| `chain_task_names` | array of strings | Task names to execute sequentially after this one completes, in order. |
| `chain_on_failure` | boolean | If `true`, the chain continues even when a task fails, stops, or is lost. Default: `false` (chain only on success). |

## Response

Returns the task history record created for this execution.

```json
{
  "id": 42,
  "created_at": "2026-09-08T12:00:00Z",
  "updated_at": "2026-09-08T12:00:01Z",
  "status": "pending",
  "started_at": null,
  "finished_at": null,
  "executed_by": "user123",
  "task": {
    "id": 7,
    "name": "mysql-backup-xtrabackup",
    "backend": "nomad",
    "owner": "mysql-backups"
  },
  "has_logs": false,
  "log_capture": "unknown",
  "duration": null,
  "display_name": "MySQL XtraBackup"
}
```

Use the returned `id` to [monitor the run](ref:sep-task-history).

## Example

```shell
curl -sk -X POST https://<pmm-server>/sep/execute/mysql-backup-xtrabackup \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{
       "meta": {
         "_target": "db-host-01",
         "_service_id": "svc-abc123",
         "_backup_dir": "/backups/mysql"
       }
     }'
```

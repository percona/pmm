---
title: List available tasks
slug: extensions-list-tasks
category:
  uri: extensions-tasks-api
position: 5
---

List the task definitions registered in PMM Extensions. Each task is an operation template that can be executed or scheduled.

```
GET /extensions/api/tasks/
```

Returns a paginated list of active task definitions.

## Query parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `owner` | string | Filter by task owner (for example, `mysql_backups`). |
| `target` | string | Filter by target host. |
| `backup_type` | string | Filter by backup type. |
| `limit` | integer | Results per page. |
| `offset` | integer | Pagination offset. |

## Get a single task

```
GET /extensions/api/tasks/{task_name}
```

Returns the task definition for the given task name.

## Example

```shell
curl -sk https://<pmm-server>/extensions/api/tasks/ \
     -H "Authorization: Bearer <token>"
```

To find available task names for a specific app, refer to the [Apps documentation](https://docs.percona.com/percona-monitoring-and-management/3/apps/index.html) for each installed app.

To list the executor hosts available for task execution, see [Executor hosts](ref:extensions-list-hosts).

---
title: List available tasks
slug: extensions-list-tasks
category:
  uri: extensions-tasks-api
position: 5
---

Use these endpoints to discover which apps are installed and which tasks each app provides.

## List installed apps

```
GET /extensions/api/apps/
```

Returns the list of apps currently installed in PMM Extensions, for example `mysql_backups`.

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/api/apps/ \
     -H "Authorization: Bearer <token>"
```

## List tasks for an app

```
GET /extensions/api/apps/{app}/
```

Returns the tasks created in the named app, for example the backup tasks in MySQL Backups. Results are paginated with `offset` and `limit`. Use the `app_key` value from the list above, for example `mysql_backups`. Restore tasks are under `mysql_backups/restore`.

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/api/apps/mysql_backups/ \
     -H "Authorization: Bearer <token>"
```

To find which hosts are available for task execution, [list executor hosts](ref:extensions-list-hosts).

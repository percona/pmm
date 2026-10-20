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

Returns the tasks registered by the named app. Use the `app` value from the list above, for example `mysql_backups`.

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/api/apps/mysql_backups/ \
     -H "Authorization: Bearer <token>"
```

To find which hosts are available for task execution, [list executor hosts](ref:extensions-list-hosts).

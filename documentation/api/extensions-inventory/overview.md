---
title: Overview
slug: extensions-inventory-overview
category:
  uri: extensions-inventory-api
position: 0
---

The PMM Extensions Inventory API provides access to the hosts and database services that PMM Extensions uses for task execution.

Use it to:

- [list executor hosts](ref:extensions-list-hosts) to find valid `target` values for task execution
- [list services](ref:extensions-list-services) to find database services for backup and restore tasks
- [list schemas and tables](ref:extensions-list-schemas) within a service

## Base URL

```
https://<pmm-server>/extensions
```

All endpoint paths in this reference are appended to that base.

## Authentication

Use the same PMM service account token as the Tasks API:

```shell
curl -sk https://<pmm-server>/extensions/api/extensions/hosts/ \
     -H "Authorization: Bearer <token>"
```

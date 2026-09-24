---
title: Overview
slug: extensions-inventory-overview
category:
  uri: extensions-inventory-api
position: 0
---

The PMM Extensions Inventory API provides access to the host and database inventory that PMM Extensions keeps in sync with your PMM inventory. Tasks use this inventory to know where your databases run and which hosts can reach them.

Use it to:

- [list nodes and services](ref:extensions-list-nodes) to find IDs for task execution
- [check sync status](#sync-status-fields) to see whether a node or service is in sync with PMM
- [retire and revive](ref:extensions-list-nodes) nodes, services, schemas, and tables
- [browse schemas and tables](ref:extensions-list-schemas) within a service
- [resolve duplicate records](ref:extensions-list-nodes) when PMM re-registers a node or service with a new ID

## Data model

The inventory follows a strict hierarchy:

```
Node (host)
  └── Service (database instance on that host)
        └── Schema (database within the instance)
              └── Table
```

- **Node**: a physical or virtual host, synced from PMM. Key fields: `address`, `name`, `external_id` (the PMM node ID), and `type` (always `generic` for PMM-sourced nodes).
- **Service**: a database running on a node. `type` is one of `mysql`, `postgresql`, `mongodb`, `proxysql`, `haproxy`, `external`, or `valkey`.
- **Schema**: a named database within a service.
- **Table**: a table within a schema, including its `CREATE` statement and index keys.

## Retiring vs deleting

PMM Extensions uses soft deletes. Retiring a node or service sets `retired_at` but keeps the record so you can still look it up and restore it later. Retiring a node also retires all its services, schemas, and tables. Use [revive](ref:extensions-list-nodes) to restore a retired record.

## Sync status fields

Every node and service carries sync health fields that show whether PMM Extensions is keeping up with PMM:

| Field | Description |
|-------|-------------|
| `last_synced_at` | When PMM Extensions last confirmed this record against PMM |
| `last_sync_error` | Error message from the most recent failed sync attempt, or `null` when syncing cleanly |
| `sync_failing_since` | When the current run of failures began, or `null` when healthy |
| `consecutive_failures` | Number of failed sync attempts since the last success |

## Base URL

The PMM Extensions Inventory API is proxied through PMM Server at:

```
https://<pmm-server>/extensions/
```

All endpoints in this reference are relative to that base.

## Authentication

Use the same PMM service account token as the Tasks API:

```shell
curl -sk https://<pmm-server>/extensions/nodes/ \
     -H "Authorization: Bearer <token>"
```

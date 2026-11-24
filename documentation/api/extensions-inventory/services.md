---
title: Services and schemas
slug: extensions-list-services
category:
  uri: extensions-inventory-api
position: 2
---

Services represent database instances in your PMM inventory. Use these endpoints to find service IDs for backup and restore tasks.

## List services

```
GET /extensions/api/extensions/services/
```

Returns a paginated list of database services.

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `service_type` | string | none | Filter by type: `mysql`, `postgresql`, `mongodb`, `proxysql`, `haproxy`, `external`, `valkey` |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page |

**Example — list all MySQL services:**

```shell
curl -sk "https://<pmm-server>/extensions/api/extensions/services/?service_type=mysql" \
     -H "Authorization: Bearer <token>"
```

## List schemas for a service

```
GET /extensions/api/extensions/services/{service_id}/schemas
```

Returns the schemas (databases) within a service. Use `search` to filter by name.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `search` | string | Case-insensitive name filter |

**Example:**

```shell
curl -sk "https://<pmm-server>/extensions/api/extensions/services/7/schemas?search=prod" \
     -H "Authorization: Bearer <token>"
```

---
title: Tables
slug: extensions-list-schemas
category:
  uri: extensions-inventory-api
position: 3
---

Returns the tables within a schema. Use this to populate table selectors in schema-aware task forms.

## List tables for a schema

```
GET /extensions/api/extensions/schemas/{schema_id}/tables
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `search` | string | Case-insensitive name filter |

**Example:**

```shell
curl -sk "https://<pmm-server>/extensions/api/extensions/schemas/12/tables" \
     -H "Authorization: Bearer <token>"
```

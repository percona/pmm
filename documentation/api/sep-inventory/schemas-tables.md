---
title: Schemas and tables
slug: sep-list-schemas
category:
  uri: sep-inventory-api
position: 3
---

Schemas represent named databases within a service. Tables are synced within each schema, including their `CREATE` statement and index keys. This data is used by apps that need to inspect or act on database structure, such as schema change operations.

## List schemas

```
GET /schemas/
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `search` | string | none | Case-insensitive search |
| `include_retired` | boolean | `false` | Include retired schemas |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `name` | Sort key. Prefix with `-` for descending. Options: `created_at`, `name`, `service_id` |

## List schemas for a service

```
GET /services/{service_id}/schemas/
```

Returns schemas for a specific service. Pass `include_tables=1` to include table records in each schema response.

**Example:**

```shell
curl -sk "https://<pmm-server>/sep-inventory/services/7/schemas/?include_tables=1" \
     -H "Authorization: Bearer <token>"
```

## Get a schema

```
GET /schemas/{schema_id}
```

Returns the schema with its tables and parent service.

## Retire a schema

```
DELETE /schemas/{schema_id}
```

Retires the schema and its tables. Returns HTTP 204.

## Revive a schema

```
POST /schemas/{schema_id}/revive
```

Restores the schema and its retired parent service and node.

## List tables

```
GET /tables/
```

Lists tables across all schemas. Accepts the same pagination and sort parameters as the schema list, with `schema_id` available as an additional sort key.

## List tables for a schema

```
GET /schemas/{schema_id}/tables/
```

Returns tables within a specific schema.

## Get a table

```
GET /tables/{table_id}
```

Returns the table with its parent schema. The response includes:

- `name`: table name
- `create`: the `CREATE TABLE` statement
- `keys`: index definitions
- `schema_id`: the parent schema ID

## Retire a table

```
DELETE /tables/{table_id}
```

Retires the table. Returns HTTP 204.

## Revive a table

```
POST /tables/{table_id}/revive
```

Restores the table and any retired ancestors (schema, service, node).

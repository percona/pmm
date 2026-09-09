---
title: Services
slug: sep-list-services
category:
  uri: sep-inventory-api
position: 2
---

Services represent database instances running on nodes. Each service carries a type, port, PMM service ID (`external_id`), and sync health fields.

Supported service types: `mysql`, `postgresql`, `mongodb`, `proxysql`, `haproxy`, `external`, `valkey`.

## List services

```
GET /services/
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `external_id` | string | none | Return only the service with this PMM service ID |
| `service_type` | string | none | Filter by type: `mysql`, `postgresql`, `mongodb`, `proxysql`, `haproxy`, `external`, `valkey` |
| `search` | string | none | Case-insensitive search across service names and other searchable columns |
| `include_retired` | boolean | `false` | Include retired services |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `name` | Sort key. Prefix with `-` for descending. Options: `created_at`, `name` |

**List all MySQL services:**

```shell
curl -sk "https://<pmm-server>/sep/services/?service_type=mysql" \
     -H "Authorization: Bearer <token>"
```

**Look up a service by PMM service ID:**

```shell
curl -sk "https://<pmm-server>/sep/services/?external_id=svc-abc123" \
     -H "Authorization: Bearer <token>"
```

## List services for a node

```
GET /nodes/{node_id}/services/
```

Returns services on a specific node. Accepts the same query parameters as the global list.

## Get a service

```
GET /services/{service_id}
```

Returns the full record for a service, including its schemas and the parent node.

## Retire a service

```
DELETE /services/{service_id}
```

Retires the service and all its schemas and tables. Returns HTTP 204.

## Revive a service

```
POST /services/{service_id}/revive
```

Restores a retired service and its parent node if also retired. Returns HTTP 409 if an active service already uses the same identifier.

## Duplicate service records

Same pattern as [duplicate node records](ref:sep-list-nodes). Use when PMM re-registration creates a second service record for the same database instance.

### List duplicate candidates

```
GET /services/identity-candidates
```

### Resolve a duplicate

```
POST /services/{service_id}/identity-link
```

Request body is identical to the node endpoint:

```json
{
  "successor_id": 99,
  "decision": "confirmed"
}
```

### List service ID history

```
GET /services/{service_id}/identity-aliases
```

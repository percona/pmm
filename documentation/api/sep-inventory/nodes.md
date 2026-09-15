---
title: Nodes
slug: sep-list-nodes
category:
  uri: sep-inventory-api
position: 1
---

Nodes represent hosts in your infrastructure, synced from PMM. Each node carries the host address, PMM node ID (`external_id`), and sync health fields.

## List nodes

```
GET /nodes/
```

**Query parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `external_id` | string | none | Return only the node with this PMM node ID |
| `source` | string | none | Filter by source. Only `pmm` is currently supported |
| `node_type` | string | none | Filter by node type |
| `search` | string | none | Case-insensitive search across node names and other searchable columns |
| `include_retired` | boolean | `false` | Include retired nodes |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `name` | Sort key. Prefix with `-` for descending. Options: `created_at`, `name` |

**Example:**

```shell
curl -sk "https://<pmm-server>/sep/nodes/?search=db-host" \
     -H "Authorization: Bearer <token>"
```

## Get a node

```
GET /nodes/{node_id}
```

Returns the full record for a node, including its services.

## Get inventory summary

```
GET /summary/
```

Returns how many nodes, services, schemas, and tables are currently registered in the inventory.

## Retire a node

```
DELETE /nodes/{node_id}
```

Retires the node and all its services, schemas, and tables. The records are kept and can be restored. Returns HTTP 204.

## Revive a node

```
POST /nodes/{node_id}/revive
```

Restores a retired node. Its services stay retired. Revive them separately if needed. Returns HTTP 409 if an active node already uses the same identifier.

## Duplicate node records

When PMM re-registers a host with a new node ID, SEP may create a second node record for the same physical host. The identity endpoints let you identify and resolve these duplicates.

### List duplicate candidates

```
GET /nodes/identity-candidates
```

Lists node pairs that may represent the same physical host. Each result shows both records and the fields they matched on.

### Resolve a duplicate

```
POST /nodes/{node_id}/identity-link
```

Pass the ID of the original node in the path. Supply your decision in the request body:

```json
{
  "successor_id": 42,
  "decision": "confirmed"
}
```

| Decision | Effect |
|----------|--------|
| `confirmed` | Merges the new record into the original. Future syncs treat them as the same node. |
| `rejected` | Records that the pairing was reviewed and is not a match. |
| `reversed` | Undoes a previous confirmation. |

### List node ID history

```
GET /nodes/{node_id}/identity-aliases
```

Lists all PMM node IDs this node has been associated with, oldest first. Useful for auditing how a host's identity has changed over time.

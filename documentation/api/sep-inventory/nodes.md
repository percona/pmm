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
| `external_id` | string | — | Return only the node with this PMM node ID |
| `source` | string | — | Filter by source. Currently only `pmm` is supported |
| `node_type` | string | — | Filter by node type |
| `search` | string | — | Case-insensitive search across node names and other searchable columns |
| `include_retired` | boolean | `false` | Include retired (soft-deleted) nodes |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 50 | Results per page (max 200) |
| `sort` | string | `name` | Sort key. Prefix with `-` for descending. Options: `created_at`, `name` |

**Example:**

```shell
curl -sk "https://<pmm-server>/sep-inventory/nodes/?search=db-host" \
     -H "Authorization: Bearer <token>"
```

## Get a node

```
GET /nodes/{node_id}
```

Returns the full record for a node, including its nested services.

## Get inventory summary

```
GET /summary/
```

Returns a count of each entity type in the inventory — a quick snapshot of how many nodes, services, schemas, and tables are registered.

## Retire a node

```
DELETE /nodes/{node_id}
```

Soft-deletes the node and cascades to all its services, schemas, and tables. The rows remain resolvable. Returns HTTP 204.

## Revive a node

```
POST /nodes/{node_id}/revive
```

Restores a retired node. Its services remain retired — revive them separately if needed. Returns HTTP 409 if an active node already holds the same unique key.

## Node identity management

When PMM re-registers a host with a different node ID, SEP may detect the predecessor and successor as separate nodes. The identity endpoints let you resolve the pairing.

### List identity candidates

```
GET /nodes/identity-candidates
```

Lists node pairs that a PMM re-registration may have split. Each result shows the `predecessor`, `successor`, and the fields they `matched_on`.

### Confirm, reject, or reverse a pairing

```
POST /nodes/{node_id}/identity-link
```

The path identifies the predecessor. Supply your decision in the request body:

```json
{
  "successor_id": 42,
  "decision": "confirmed"
}
```

| Decision | Effect |
|----------|--------|
| `confirmed` | Merges successor into predecessor; future syncs treat them as the same node |
| `rejected` | Records that the pairing was reviewed and is not a match |
| `reversed` | Undoes a previous confirmation |

### List identity aliases

```
GET /nodes/{node_id}/identity-aliases
```

Lists all upstream identifiers this node has answered for, oldest first. Useful for auditing which PMM node IDs have been associated with a given SEP node over time.

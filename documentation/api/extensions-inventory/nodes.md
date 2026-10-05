---
title: Executor hosts
slug: extensions-list-hosts
category:
  uri: extensions-inventory-api
position: 1
---

Returns the executor hosts available for task execution, enriched with display names from PMM inventory.

## List executor hosts

```
GET /extensions/api/extensions/hosts/
```

Returns a list of all executor hosts. Use the `id` field as the `target` when executing a task.

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Executor node name. Use this as `target` in execute requests. |
| `name` | string | Human-readable label from PMM inventory, or the executor node name if no inventory match exists. |
| `address` | string | Network address reported by the executor. |
| `can_elevate` | boolean or null | Whether `sudo` is available on this host. `null` if never observed. |

**Example:**

```shell
curl -sk https://<pmm-server>/extensions/api/extensions/hosts/ \
     -H "Authorization: Bearer <token>"
```

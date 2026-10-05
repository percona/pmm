---
title: Check service connectivity
slug: extensions-connectivity-check
category:
  uri: extensions-tasks-api
position: 4
---

Probes the inter-service connections PMM Extensions depends on and reports their status. Admin only.

```
POST /extensions/api/extensions/admin/connectivity-check/
```

Runs the selected probes concurrently and returns one result per requested service. Always returns HTTP 200; failures are reported in the result, not as HTTP errors.

## Request body

```json
{
  "targets": ["pmm", "inventory", "tasks", "nomad", "delivery"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `targets` | array of strings | Services to probe. At least one required. Duplicates are ignored. |

**Supported targets:**

| Value | What is probed |
|-------|----------------|
| `pmm` | PMM Server API |
| `inventory` | Inventory service |
| `tasks` | Tasks service |
| `nomad` | Nomad executor backend (probed via the Tasks service) |
| `delivery` | Diagnostics delivery receiver |

## Response

Returns a list of connectivity results in the same order as `targets`.

```json
[
  {"service": "tasks", "status": "reachable"},
  {"service": "nomad", "status": "reachable"}
]
```

**Status values:** `reachable`, `unreachable`, `not_configured`, `inputs_drifted`, `probe_undeclared`.

## Example

```shell
curl -sk -X POST https://<pmm-server>/extensions/api/extensions/admin/connectivity-check/ \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"targets": ["tasks", "nomad"]}'
```

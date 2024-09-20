---
title: List Nodes
slug: listnodes
categorySlug: inventory-api
---

## List Nodes

This section describes how to list PMM Inventory Nodes.

Example:

```shell
curl --insecure -X POST \
  --header 'Authorization: Bearer XXXXX' \
	--header 'Accept: application/json' \
	--url https://127.0.0.1/v1/inventory/nodes?node_type=NODE_TYPE_GENERIC_NODE
```

To get the authentication token, check [Authentication](ref:authentication).

Then, choose one of the following Node types:

- NODE_TYPE_GENERIC_NODE
- NODE_TYPE_CONTAINER_NODE
- NODE_TYPE_REMOTE_NODE
- NODE_TYPE_REMOTE_RDS_NODE
- NODE_TYPE_REMOTE_AZURE_DATABASE_NODE`

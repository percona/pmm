---
title: List Nodes
slug: listnodes
category: 66aca9bf17142b005ad4e9fa
---

## List Nodes

This section describes how to list PMM Inventory Nodes.

Example:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Nodes/List \
	--data '{"node_type": "GENERIC_NODE"}'
```

First, get the [authentication token](ref:authentication).

Then, choose from the following Node types:

- GENERIC_NODE
- CONTAINER_NODE
- REMOTE_NODE
- REMOTE_RDS_NODE
- REMOTE_AZURE_DATABASE_NODE`

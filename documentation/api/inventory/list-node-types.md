---
title: List Node Types
slug: listnodetypes
category:
  uri: inventory-api
---

## List Node Types

The following API call lists the Node types present in the inventory, without listing the Nodes themselves:

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/nodes:getTypes
```

To get the authentication token, check [Authentication](ref:authentication).

Unlike [List Nodes](ref:listnodes), which requires the Admin role, this endpoint is available to the Viewer role.

The response holds each Node type once:

```json
{
  "node_types": [
    "NODE_TYPE_GENERIC_NODE",
    "NODE_TYPE_REMOTE_RDS_NODE"
  ]
}
```

The possible values are:

- NODE_TYPE_GENERIC_NODE
- NODE_TYPE_CONTAINER_NODE
- NODE_TYPE_REMOTE_NODE
- NODE_TYPE_REMOTE_RDS_NODE
- NODE_TYPE_REMOTE_AZURE_DATABASE_NODE

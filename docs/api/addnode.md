---
slug: 'addnode'
---

## Add a Node

This section describes how to add a Node of any type to the inventory.

Example:

```bash
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Nodes/Add \
	--data '
{
  "node_type": "GENERIC_NODE",
  "generic": {
    "node_name": "mysql-sales-db-prod-1",
    "region": "us-east-1",
    "az": "us-east-1a",
    "address":  "209.0.25.100",
    "custom_labels": {
      "environment": "sales-prod",
      "region":  "test-region"
    }
  }
}
'
```

First, get the [authentication token](ref:authentication).

Then, choose from the following Node types:

- GENERIC_NODE
- CONTAINER_NODE
- REMOTE_NODE
- REMOTE_RDS_NODE
- REMOTE_AZURE_DATABASE_NODE

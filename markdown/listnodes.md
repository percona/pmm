---
slug: 'listnodes'
---

## List Nodes

This section describes listing the Nodes in the inventory.

Example:
```bash
curl --insecure -X POST -H 'Authorization: Bearer XXXXX'      
	--reuest POST      
	--url https://127.0.0.1/v1/inventory/Nodes/List
	--header 'Accept: application/json'
	--header 'Content-Type: application/json'
	--data '
{
     "node_type": "GENERIC_NODE"
}
'
```
Firstly, get the [authentication string](ref:authentication).

Then, choose from the following Node types: 
`NODE_TYPE_INVALID, GENERIC_NODE, CONTAINER_NODE, REMOTE_NODE, REMOTE_RDS_NODE, REMOTE_AZURE_DATABASE_NODE`





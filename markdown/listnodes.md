---
slug: 'listnodes'
---

## List Nodes

In this section we are going to describe how can you list Nodes in the inventory.

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
First you have to get the [authetication string](ref:authentication).

Then you have to choose form the follwing Node types, usually `GENERIC_NODE` is a good first choose:
`NODE_TYPE_INVALID, GENERIC_NODE, CONTAINER_NODE, REMOTE_NODE, REMOTE_RDS_NODE, REMOTE_AZURE_DATABASE_NODE`





---
title: Add a Node
slug: addnode
categorySlug: inventory-api
---

## Add a Node

This section describes how to add a Node of any type to PMM Inventory.

In PMM version 2, we featured a separate API call for each Node type. Starting with PMM 3.0.0, we have streamlined the process by offering a single API endpoint for all Node types. 

Previously, the Node type was defined by the endpoint, i.e. `Nodes/AddGeneric`. In the new approach, the Node type must be specified as the top-level property of the request payload. As part of this single API endpoint update, we have deprecated individual API endpoint for each Node type.

Here's how to add a Node of type `generic` using the old and new API calls.

**Old API call**
  
```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/Nodes/AddGeneric \
     --data '
{
  "node_name": "mysql-sales-db-prod-1",
  "region": "us-east-1",
  "az": "us-east-1a",
  "address":  "209.0.25.100",
  "environment": "sales-prod",
  "custom_labels": {
    "department":  "sales"
  }
}
'
```

**New API call**

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/nodes \
     --data '
{
  "generic": {
    "node_name": "mysql-sales-db-prod-1",
    "region": "us-east-1",
    "az": "us-east-1a",
    "address":  "209.0.25.100",
    "custom_labels": {
      "environment": "sales-prod",
      "department":  "sales"
    }
  }
}
'
```

You can choose from the following Node types:

- generic
- container
- remote
- remote_rds
- remote_azure

To get the authentication token, check [Authentication](ref:authentication).

---
title: Add a Node
slug: addnode
category: 626de009b977e3003179f7dd
---

## Add a Node

This section describes how to add a Node of any type to PMM Inventory.

In PMM 2, we featured a separate API call for each Node type. Starting with PMM 3, we offer a single API endpoint for all Node types. While previously the Node type was defined by the endpoint, i.e. `Nodes/AddGeneric`, now the Node type must be specified as the top-level property of the request payload. Along with this single API endpoint, we are deprecating the separate API calls for each Node type.

Let's see how to add a Node of type `generic` using the old and new API calls.

Old API call:
  
```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
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

New API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Nodes/Add \
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

To get the authentication token, please visit [this page](ref:authentication).

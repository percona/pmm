---
title: Add a Node
slug: addnode
category: 626de009b977e3003179f7dd
---

## Add a Node

This section describes how to add a Node of any type to the inventory.

In PMM versions prior to 2.40.0, we featured a separate API call for each Node type. Starting with PMM 2.40.0, we have a single API call for all Node types. The API call is `Add` and the Node type is specified in the `node_type` field. The `node_type` field is required. Along with this single API endpoint, we are deprecating the separate API calls for each Node type.

Let's see how to add a Node of type `GENERIC_NODE` using the old and new API calls.

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
  "custom_labels": {
    "environment": "sales-prod",
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

To get the authentication token, please visit [this page](ref:authentication).

You can choose from the following Node types:

- GENERIC_NODE: `generic`
- CONTAINER_NODE: `container`
- REMOTE_NODE: `remote`
- REMOTE_RDS_NODE: `remote_rds`
- REMOTE_AZURE_DATABASE_NODE: `remote_azure`

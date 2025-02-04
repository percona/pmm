---
title: Change Agent Attributes
slug: changeagent
categorySlug: inventory-api
---

## Change Agent Attributes

This section describes how to change Agent attributes.

> 📘 Info
> 
> Not all attributes can be changed. For example, you cannot change the Agent type nor its ID.

In PMM versions prior to 3.0.0, we featured a separate API call for each Agent type. Starting with PMM 3.0.0, we have streamlined the process by offering a single API endpoint for all Agent types. 

Previously, the Agent type was defined by the endpoint, i.e. `/v1/inventory/Agents/ChangeMySQLdExporter`. In the new approach, the Agent type must be specified as the top-level property of the request payload. 
As part of the single API endpoint update, we have deprecated individual API endpoints for each Agent type.

Here's how to change Agent attributes using the old and new API calls. The goal of this sample API call is to enable the Agent, make it work in Push mode, and assign a few custom labels to it.

**Old API call**

```shell
curl --insecure -X POST \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/Agents/ChangeMySQLdExporter \
     --data '
{
  "agent_id": "13519ec9-eedc-4d21-868c-582e146e1d0e",
  "common": {
    "enable":  true,
    "custom_labels": {
      "department":  "sales",
      "replication_set": "db-sales-prod-1-rs1",
    },
    enable_push_metrics: true,
  }
}
'
```

**New API call**

```shell
curl --insecure -X PUT \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Content-Type: application/json' \
     --url https://127.0.0.1/v1/inventory/agents/13519ec9-eedc-4d21-868c-582e146e1d0e \
     --data '
{
  "mysqld_exporter": {
    "enable":  true,
    "enable_push_metrics": true,
    "custom_labels": {
      "department":  "sales",
      "replication_set": "db-sales-prod-1-rs1"
    }
  }
}
'
```

To get the authentication token, check [Authentication](ref:authentication).

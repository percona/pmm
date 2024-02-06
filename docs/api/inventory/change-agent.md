---
title: Change Agent Attributes
slug: changeagent
category: 626de009b977e3003179f7dd
---

## Change Agent Attributes

This section describes how to change Agent attributes.

Please note that not all attrbutes can be chaged. For example, you cannot change the Agent type nor its ID.

In PMM versions prior to 3.0.0, we featured a separate API call for each Agent type. Starting with PMM 3.0.0, we offer a single API endpoint for all Agent types. While previously the Agent type was defined by the endpoint, i.e. `/v1/inventory/Agents/ChangeMySQLdExporter`, now the Agent type must be specified as the top-level property of the request payload. Along with this single API endpoint, we are deprecating individual API endpoints for each Agent type.

Let's see how to change Agent attributes using the old and new API calls. The goal of this sample API call is to enable the agent, make it work in push mode, and remove a few custom labels from it. 

Old API call:

```shell
curl --insecure -X POST \
  -H 'Authorization: Basic YWRtaW46YWRtaW4=' \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	--url https://127.0.0.1/v1/inventory/Agents/ChangeMySQLdExporter \
	--data '
{
  "agent_id": "/agent_id/13519ec9-eedc-4d21-868c-582e146e1d0e",
  "common": {
    "enable":  true,
    "custom_labels": {
      "department":  "sales",
      "replication_set": "db-sales-prod-1-rs1",
    },
    remove_custom_labels: true,
    enable_push_metrics: true,
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
	--url https://127.0.0.1/v1/inventory/Agents/Add \
	--data '
{
  "mysqld_exporter": {
    "agent_id": "/agent_id/13519ec9-eedc-4d21-868c-582e146e1d0e",
    "common": {
      "enable":  true,
      "custom_labels": {
        "department":  "sales",
        "replication_set": "db-sales-prod-1-rs1",
      },
      remove_custom_labels: true,
      enable_push_metrics: true,
    }
  }
}
'
```

To get the authentication token, please visit [this page](ref:authentication).

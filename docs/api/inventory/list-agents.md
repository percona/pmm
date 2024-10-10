---
title: List Agents
slug: listagents
categorySlug: inventory-api
---

### Get all agents running on a specific node

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/agents?node_id=7d07a712-7fb9-4265-8a7d-a0db8aa35762
```

### Get all agents providing insights to a specific service

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/agents?service_id=6ed34dba-1f4a-4cb7-b02f-49aac5c46004
```

### Get all agents of a specific type

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/agents?agent_type=AGENT_TYPE_MYSQLD_EXPORTER
```

### Get all agents started by a specific PMM Agent

```shell
curl --insecure -X GET \
     --header 'Authorization: Bearer XXXXX' \
     --header 'Accept: application/json' \
     --url https://127.0.0.1/v1/inventory/agents?pmm_agent_id=a05941d3-90cb-49e1-ba85-f765de408fee
```

To get the authentication token, check [Authentication](ref:authentication).

If you need to know the `node_id`, refer to [List Nodes endpoint](ref:listnodes).

For the `agent_type` parameter, you can check the options provided by the respective query param further down this page.

> 🚧 Important
> 
> Exactly one of the following parameters is required for the endpoint to succeed: `pmm_agent_id`, `node_id`, or `service_id`.

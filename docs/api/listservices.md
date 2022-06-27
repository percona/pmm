---
slug: 'listservices'
---

## List Services

The following Curl API call lists the available services on a Node:

```bash
curl --insecure -X POST -H 'Authorization: Bearer XXXXX' \
     --request POST \
     --url https://127.0.0.1/v1/inventory/Services/List \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json' \
     --data '
{
  "node_id": "/node_id/XXXXX",
  "service_type": "MYSQL_SERVICE"
}'
```

Firstly, get the [authentication string](ref:authentication).

Then, you require the [node_id](ref:listnodes).

Choose the `service_type` that you want to list. The options are:
`SERVICE_TYPE_INVALID, MYSQL_SERVICE, MONGODB_SERVICE, POSTGRESQL_SERVICE, PROXYSQL_SERVICE, HAPROXY_SERVICE, EXTERNAL_SERVICE`



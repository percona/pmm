---
slug: 'listservices'
---

## List Services

The Following Curl API call will list the available services on a Node:

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

First you have to get the [authetication string](ref:authentication).

Then you need the [node_id](ref:listnodes).

You have to also choose which `service_type` you would like to list, the options are:
`SERVICE_TYPE_INVALID, MYSQL_SERVICE, MONGODB_SERVICE, POSTGRESQL_SERVICE, PROXYSQL_SERVICE, HAPROXY_SERVICE, EXTERNAL_SERVICE`



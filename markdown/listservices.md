---
slug: 'listservices'
---

## List Services

The Following Curl API call will list the available services on a Node:

```
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

First you have to get the (<a href="https://percona-pmm.readme.io/reference/authentication">authetication string</a>).

Then you need the (<a href="https://percona-pmm.readme.io/reference/listnodes">node_id</a>).

You have to also choose which `service_type` you would like to list, the options are:
`SERVICE_TYPE_INVALID, MYSQL_SERVICE, MONGODB_SERVICE, POSTGRESQL_SERVICE, PROXYSQL_SERVICE, HAPROXY_SERVICE, EXTERNAL_SERVICE`



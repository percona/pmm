---
title: Create a PSMDB Cluster
slug: create-psmdb-cluster
category: 651c00ce1679590036133c8c
order: 2
hidden: 0
---

The PSMDB Create endpoint receives this structure of:

```json
{
  "kubernetes_cluster_name": "string",
  "name": "string",
  "params": {
    "cluster_size": 0,
    "replicaset": {
      "compute_resources": {
        "cpu_m": 0,
        "memory_bytes": "string"
      },
      "disk_size": "string"
    },
    "image": "string"
  },
  "expose": true
}
```

## Fields description

| Field                                     | Description                           | Notes                                                        |
| ----------------------------------------- | ------------------------------------- | ------------------------------------------------------------ |
| kubernetes_cluster_name                   | Kubernetes cluster name               | Required                                                     |
| name                                      | PSMDB cluster name                    | Default: `psmdb`+DB version+5 chars random string            |
| cluster_size                              | Cluster size                          | Default: 3                                                   |
| replicaset.compute_resources.cpu_m        | CPU resources millis                  | Default: 1000                                                |
| replicaset.compute_resources.memory_bytes | Max memory size in bytes              | Default: 2 GB                                                |
| disk_size                                 | Max disk size                         | Default: 25 GB                                               |
| image                                     | PSMDB Docker image                    | Default is the recommended version from the Percona's version service |
| expose                                    | Expose outside the Kubernetes cluster | Default: false                                               |


Since the endpoint can set defaults, you can create a PSMDB cluster with a minimum request like this:

```shell
curl -X POST "http://localhost/v1/management/DBaaS/PSMDBCluster/Create" \
    -H "accept: application/json" \
    -H "authorization: Basic YWRtaW46YWRtaW4=" \
    -H "Content-Type: application/json" \
    -d "{ \"kubernetes_cluster_name\": \"my_cluster\" }"
```


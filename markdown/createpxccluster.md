### Create PXC Cluster

Once we registered kubernetes cluster we can use it’s name to create DB Clusters. Here is an example for PXC Cluster, the values for parameters are recomended by Percona:

```bash
curl -X POST "http://localhost/v1/management/DBaaS/PXCCluster/Create" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\", \"expose\": true, \"params\": { \"cluster_size\": 3, \"pxc\": { \"compute_resources\": { \"cpu_m\": 1000, \"memory_bytes\": 2000000000 }, \"disk_size\": 25000000000, \"image\": \"percona/percona-xtradb-cluster:8.0.25-15.1\" }, \"haproxy\": { \"compute_resources\": { \"cpu_m\": 1000, \"memory_bytes\": 2000000000 } } } }"
```

### Request parameters

```
{
  "kubernetes_cluster_name": "string",
  "name": "string",
  "params": {
    "cluster_size": 0,
    "pxc": {
      "image": "string",
      "compute_resources": {
        "cpu_m": 0,
        "memory_bytes": "string"
      },
      "disk_size": "string"
    },
    "proxysql": {
      "image": "string",
      "compute_resources": {
        "cpu_m": 0,
        "memory_bytes": "string"
      },
      "disk_size": "string"
    },
    "haproxy": {
      "image": "string",
      "compute_resources": {
        "cpu_m": 0,
        "memory_bytes": "string"
      }
    }
  },
  "expose": true
}
```

#### Fields description

|Parameter                              |Description                                     |Notes                                                                |
|---------------------------------------|------------------------------------------------|---------------------------------------------------------------------|
|kubernetes_cluster_name                |Kubernetes cluster name                         |Required                                                             |
|name                                   |PXC cluster name to create                      |Default: pxc + DB version + 5 chars random string                    |
|cluster_size                           |Cluster size                                    |Default: 3                                                           |
|image                                  |Docker image name                               |Default is the recommended version from the Percona's version service|
|compute_resources.cpu_m                |CPU resources millis                            |Default: 1000                                                        |
|mcompute_resources.memory_bytes        |Max memory size in bytes                        |Default: 2 Gb                                                        |
|disk_size                              |Max disk size for the PXC instance              |Default: 25 Gb                                                       |
|proxysql.image                         |Docker image for ProxySQL                       |Default: empty. (Use operator's default)                             |
|proxysql.compute_resources.cpu_m       |CPU resources millis                            |Default: 1000                                                        |
|proxysql.compute_resources.memory_bytes|Max memory size in bytes                        |Default 2 Gb                                                         |
|proxysql.disk_size                     |Max disk size for ProxySQL                      |Default: empty, use operator's default                               |
|haproxyimage                           |Docker image for HA Proxy                       |Default: empty, use operator's default                               |
|haproxy.compute_resources.cpu_m        |CPU resources millis                            |Default: 1000                                                        |
|haproxy.compute_resources.memory_bytes |Max memory size in bytes                        |Default: 2 Gb                                                        |
|expose                                 |Make it available outside the Kubernetes cluster|Default: false                                                       |

**Note:** 
Only one of ProxySQL or HAProxy should be specified in the request.

#### Minimum request example

Since the API has the defaults mentioned above, the HTTP request can have the kubernetes cluster name as the only parameter.

Example:

```bash
curl -X POST "http://localhost/v1/management/DBaaS/PXCCluster/Create" \
    -H "accept: application/json" \
    -H "authorization: Basic YWRtaW46YWRtaW4=" \
    -H "Content-Type: application/json" \
    -d '{ "kubernetes_cluster_name": "my_cluster" }'
```

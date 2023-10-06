---
title: Private DBaaS
slug: dbaas-overview
category: 651c00ce1679590036133c8c
order: 0
hidden: 0
---

## Private DBaaS
Database-as-a-Service (DBaaS) is a managed database that doesn’t need to be installed and maintained but is instead provided as a service to the user. 

A common misconception is that a DBaaS is limited to the public cloud. As many enterprises already have large data centers and heavy investments in hardware, an on-premise DBaaS can also be quite appealing. Keeping the database in-house is often favored when the hardware and resources are already available. In addition, there are extra compliance and security concerns when looking at a public cloud offering.

The [Percona Monitoring and Management](https://www.percona.com/software/database-tools/percona-monitoring-and-management) (PMM) DBaaS component is a private DBaaS that simplifies and automates [Percona kubernetes operators](https://www.percona.com/software/percona-kubernetes-operators) to created DBs in a Kubernetes cluster.

It creates and manages DBs such as [Percona XtraDB Cluster](https://www.percona.com/doc/kubernetes-operator-for-pxc/index.html) (PXC) and [Percona Server for MongoDB](https://www.percona.com/doc/kubernetes-operator-for-psmongodb/index.html) (PSMDB) and automates tasks such as:
  - Installing the database software
  - Configuring the database
  - Setting up backups
  - Managing upgrades
  - Handling failover scenarios

Read more about DBaaS:
- [DBaaS Documentation](https://docs.percona.com/percona-monitoring-and-management/setting-up/server/dbaas.html)
- [DBaaS blogs](https://www.percona.com/blog/tag/dbaas/)

## How to configure and use

To configure and use DBaaS you would need to have PMM deployment and Kubernetes cluster. PMM provides functionality to create and manage DBs and Kubernetes cluster is where those DBs will be running.

How to get this environment up and running you can read in our [documentation](https://docs.percona.com/percona-monitoring-and-management/setting-up/server/dbaas.html#create-a-kubernetes-cluster).

PMM provides set of API calls to enable DBaaS, configure it and to create and manage DBs:
- [Change Settings](ref:changesettings)
- [Register Kubernetes Cluster](ref:registerkubernetescluster)
- [Create PXC Cluster](ref:createpxccluster)
- [Create PSMDB Cluster](ref:createpsmdbcluster)

In this example we would use minikube for the kubernetes cluster and will create a PXC DB cluster, but similar API endpoints exist for the PSMDB.

### Enabling

To enable DBaaS, first we need:
- Enable DBaaS in settings.
- Specify the DNS name or public IP address of the pmm-server instance to be able to monitor DB clusters we create in DBaaS and Kubernetes cluster itself.

It is highly recommended to **use DNS name** instead of IP address but in example bellow we have a dev environment and use IP address instead.

#### Get Docker container IP and set it as public address

First of all we should get IP address of PMM (or DNS name should be used and that is recommended). If you are running in local minikube environment you can use following command to get IP address:
```shell
IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' pmm-server)
```
If your kubernetes cluster is located outside of your local system you can get public address by calling `ifconfig`.

Then to enable DBaaS send request to `Settings/Change` endpoint like below where `IP` is public IP address or DNS name of PMM Server instance.
```shell
curl -X POST "http://localhost/v1/Settings/Change" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \
     -H "Content-Type: application/json" \ 
     -d "{ \"pmm_public_address\": \"${IP}\", \"enable_dbaas\": true }"
```

API endpoint used in this step: [Change settings](ref:changesettings).

### Registering new Kubernetes cluster

Once kubernetes cluster is created it should be registered in PMM where `my_cluster` is a name of kubernetes cluster which will be used later. `sed` command is used to remove newlines, otherwise this script doesn’t work.
```shell
KUBECONFIG=$(kubectl config view --flatten --minify | sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/\\n/g')

curl -X POST "http://localhost/v1/management/DBaaS/Kubernetes/Register" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"kube_auth\": { \"kubeconfig\": \"${KUBECONFIG}\" }}"
```
This command will register kubernetes cluster, start monitoring of kubernetes cluster and install required kubernetes operators.

API endpoint used in this step: [RegisterKubernetesCluster](ref:registerkubernetescluster)

### Get available PXC image names

To create a PXC cluster, we need to provide the image name for the database instance.
Percona maintains a list of available versions for each component. For example, to retrieve the list of the available PXC components we can call the `Components/GetPXC` API method:

```shell
curl -X POST "http://localhost/v1/management/DBaaS/Components/GetPXC" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \ 
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\"}"
```
Example response: 

```json
{
  "versions": [
    {
      "product": "pxc-operator",
      "operator": "1.10.0",
      "matrix": {
        "pxc": {
          "8.0.19-10.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.19-10.1",
            "image_hash": "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a",
            "status": "available"
          },
          "8.0.20-11.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.20-11.1",
            "image_hash": "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349",
            "status": "available"
          },
          "8.0.20-11.2": {
            "image_path": "percona/percona-xtradb-cluster:8.0.20-11.2",
            "image_hash": "feda5612db18da824e971891d6084465aa9cdc9918c18001cd95ba30916da78b",
            "status": "available"
          },
          "8.0.21-12.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.21-12.1",
            "image_hash": "d95cf39a58f09759408a00b519fe0d0b19c1b28332ece94349dd5e9cdbda017e",
            "status": "available"
          },
          "8.0.22-13.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.22-13.1",
            "image_hash": "1295af1153c1d02e9d40131eb0945b53f7f371796913e64116bf2caa77dc186d",
            "status": "available"
          },
          "8.0.23-14.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.23-14.1",
            "image_hash": "8109f7ca4fc465ba862c08021df12e77b65d384395078e31e270d14b77810d79",
            "status": "available"
          },
          "8.0.25-15.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.25-15.1",
            "image_hash": "529e979c86442429e6feabef9a2d9fc362f4626146f208fbfac704e145a492dd",
            "status": "recommended",
            "default": true
          }
        },
        "pmm": {
          "2.23.0": {
            "image_path": "percona/pmm-client:2.23.0",
            "image_hash": "8fa0e45f740fa8564cbfbdf5d9a5507a07e331f8f40ea022d3a64d7278478eac",
            "status": "recommended",
            "default": true
          }
        },
        "proxysql": {
          "2.0.18": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-proxysql",
            "image_hash": "f109a62eb316732d59dd80ed0e013fc9594cbae601586b94023b8c068f7ced7b",
            "status": "available"
          },
          "2.0.18-2": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-proxysql-8.0.25",
            "image_hash": "b84701c47a11c6f5ca46481f25f1b6086c0a30014d05584c7987f1d42a17b584",
            "status": "recommended",
            "default": true
          }
        },
        "haproxy": {
          "2.3.14": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-haproxy",
            "image_hash": "2f06ac4a0f39b2c0253421c3d024291d5ba19d41e35e633ff6ddcf4ba67fd51a",
            "status": "available"
          },
          "2.3.15": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-haproxy-8.0.25",
            "image_hash": "62479be2a21192a3215f03d3f9541decd5ef1737741245ac33ee439915a15128",
            "status": "recommended",
            "default": true
          }
        },
        "backup": {
          "2.4.24": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-pxc5.7-backup",
            "image_hash": "2ff5992220ba251cf064cc2b4d5929e0fdb963db18e35d6c672f9aacb0be3bed",
            "status": "available"
          },
          "2.4.24-2": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-pxc5.7.35-backup",
            "image_hash": "ac9fcd3078107c6492c687eb98215d4e5daf27a02fb3c78ba4b9e9c01f2078b3",
            "status": "recommended"
          },
          "8.0.23": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-pxc8.0-backup",
            "image_hash": "6ab8efb3804d1e519e49ee10eb46b428a837cfdcee222cc5ae2089cc1dc02a6d",
            "status": "available"
          },
          "8.0.25": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-pxc8.0.25-backup",
            "image_hash": "c3991f0959a3b4114d7ff629d9d3cdf0dc200c58443ca8ebb1446d8b1cbe416d",
            "status": "recommended",
            "default": true
          }
        },
        "operator": {
          "1.10.0": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0",
            "image_hash": "73d2266258b700a691db6196f4b5c830845d34d57bdef5be5ffbd45e88407309",
            "status": "recommended",
            "default": true
          }
        },
        "log_collector": {
          "1.10.0": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-1-logcollector",
            "image_hash": "8f106b1e9134812b77f4e210ad0fcd7d8d3515a90fe53554d24cd49defc9e044",
            "status": "available"
          },
          "1.10.0-2": {
            "image_path": "percona/percona-xtradb-cluster-operator:1.10.0-logcollector-8.0.25",
            "image_hash": "d69dad98900532e2ad6d0bf12c34a148462816fa3ee4697e9b73efef7583901a",
            "status": "recommended",
            "default": true
          }
        }
      }
    }
  ]
}
```

From this response, choose one of the images in the `pxc` section:
```json
      "matrix": {
        "pxc": {
          "8.0.19-10.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.19-10.1",
            "image_hash": "1058ae8eded735ebdf664807aad7187942fc9a1170b3fd0369574cb61206b63a",
            "status": "available"
          },
          "8.0.20-11.1": {
            "image_path": "percona/percona-xtradb-cluster:8.0.20-11.1",
            "image_hash": "54b1b2f5153b78b05d651034d4603a13e685cbb9b45bfa09a39864fa3f169349",
            "status": "available"
          },
```

The chosen `image_path` value is the value you should provide in the next API call as the `image` field. We recommend using the one with `"status": "recommended"`
Example: `"image": "percona/percona-xtradb-cluster:8.0.19-10.1"`

API endpoint used in this step: [ChangePXCComponents](ref:changepxccomponents).

### Create PXC Cluster

Once we registered kubernetes cluster we can use it’s name to create DB Clusters. Here is an example for PXC Cluster, the values for parameters are recomended by Percona:

```shell
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


|Parameter                              |Description                                     |Notes                                                                |
|---------------------------------------|------------------------------------------------|---------------------------------------------------------------------|
|kubernetes_cluster_name                |Kubernetes cluster name                         |Required                                                             |
|name                                   |PXC cluster name to create                      |Default: pxc + DB version + 5 chars random string                    |
|cluster_size                           |Cluster size                                    |Default: 3                                                           |
|image                                  |Docker image name                               |Default is the recommended version from the Percona's version service|
|compute_resources.cpu_m                |CPU resources millis                            |Default: 1000                                                        |
|compute_resources.memory_bytes         |Max memory size in bytes                        |Default: 2 GB                                                        |
|disk_size                              |Max disk size for the PXC instance              |Default: 25 GB                                                       |
|proxysql.image                         |Docker image for ProxySQL                       |Default: empty. (Use operator's default)                             |
|proxysql.compute_resources.cpu_m       |CPU resources millis                            |Default: 1000                                                        |
|proxysql.compute_resources.memory_bytes|Max memory size in bytes                        |Default 2 GB                                                         |
|proxysql.disk_size                     |Max disk size for ProxySQL                      |Default: empty, use operator's default                               |
|haproxyimage                           |Docker image for HA Proxy                       |Default: empty, use operator's default                               |
|haproxy.compute_resources.cpu_m        |CPU resources millis                            |Default: 1000                                                        |
|haproxy.compute_resources.memory_bytes |Max memory size in bytes                        |Default: 2 GB                                                        |
|expose                                 |Make it available outside the Kubernetes cluster|Default: false                                                       |

**Notes:** 
Either ProxySQL or HAProxy should be specified in the request.
Memory bytes are strings because the parameter accepts the unit, like *1 Gi*

#### Minimum request example

Since the API has the defaults mentioned above, the HTTP request can have the Kubernetes cluster name as the only parameter.

Example:

```shell
curl -X POST "http://localhost/v1/management/DBaaS/PXCCluster/Create" \
    -H "accept: application/json" \
    -H "authorization: Basic YWRtaW46YWRtaW4=" \
    -H "Content-Type: application/json" \
    -d '{ "kubernetes_cluster_name": "my_cluster" }'
```

API endpoint used in this step: [CreatePXCCluster](ref:createpxccluster).

### List Kubernetes clusters

Once you created PXC cluster you can check the status of the cluster by calling the `List` endpoint.
```shell
curl -X POST "http://localhost/v1/management/DBaaS/DBClusters/List" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \ 
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\"}"
```

Example response:
```json
{
  "pxc_clusters": [
    {
      "name": "my-cluster-1",
      "state": "DB_CLUSTER_STATE_READY",
      "operation": {
        "finished_steps": 6,
        "total_steps": 6
      },
      "params": {
        "cluster_size": 3,
        "pxc": {
          "compute_resources": {
            "cpu_m": 1000,
            "memory_bytes": "2000000000"
          },
          "disk_size": "25000000000"
        },
        "haproxy": {
          "compute_resources": {
            "cpu_m": 1000,
            "memory_bytes": "2000000000"
          }
        }
      },
      "installed_image": "percona/percona-xtradb-cluster:8.0.25-15.1"
    }
  ]
}
```
Response contains field `state` which provides current state of DB cluster. `DB_CLUSTER_STATE_READY` means that DB cluster is ready for use.

API endpoint used in this step: [ListDBClusters](ref:listdbclusters)

### Get credentials

Once PXC Cluster is ready we can request credentials to connect to DB.

```shell
curl -X POST "http://localhost/v1/management/DBaaS/PXCClusters/GetCredentials" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\"}"
```
**Example response:**
```json
{
  "connection_credentials": {
    "username": "root",
    "password": "8fhAK0wjBLcjPncEfJM2r4Ny",
    "host": "my-cluster-1-haproxy.default",
    "port": 3306
  }
}
```

API endpoint used in this step: [GetPXCClusterCredentials](ref:getpxcclustercredentials)

### Create a PSMDB Cluster

The PSMDB `Create` endpoint can also set defaults, so creating a PSMDB cluster can be made with a request like this:

```shell
curl -X POST "http://localhost/v1/management/DBaaS/PSMDBCluster/Create" \
    -H "accept: application/json" \
    -H "authorization: Basic YWRtaW46YWRtaW4=" \
    -H "Content-Type: application/json" \
    -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"expose\": true}"
```

#### Request fields

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

| Field                                     | Description                           | Notes                                                        |
| ----------------------------------------- | ------------------------------------- | ------------------------------------------------------------ |
| kubernetes_cluster_name                   | Kubernetes cluster name               | Required                                                     |
| name                                      | PSMDB cluster name                    | Default: `psmdb`+DB version+5 chars random string            |
| cluster_size                              | Cluster size                          | Default: 3                                                   |
| replicaset.compute_resources.cpu_m        | CPU resources millis                  | Default: 1000                                                |
| replicaset.compute_resources.memory_bytes | Max memory size in bytes              | Default: 2 GB                                                |
| disk_size                                 | Max disk size                         | Default: 25 Gb                                               |
| image                                     | PSMDB Docker image                    | Default is the recommended version from the Percona's version service |
| expose                                    | Expose outside the Kubernetes cluster | Default: false                                               |

### Delete DB Cluster

If you don’t need the database cluster you can delete it using the request below.
```shell
curl -X POST "http://localhost/v1/management/DBaaS/DBClusters/Delete" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\", \"cluster_type\": \"DB_CLUSTER_TYPE_PXC\"}"
```

API endpoint used in this step: [DeleteDBCluster deletes](ref:deletedbcluster)

### Unregister Kubernetes Cluster

After we played with DBaaS we can unregister kubernetes cluster. 

Unregister a kubernetes cluster doesn’t delete anything, it just removes the cluster from the list of registered clusters and all database clusters will remain active and will send metrics to PMM.

```shell
curl -X POST "http://localhost/v1/management/DBaaS/Kubernetes/Unregister" \ 
     -H "accept: application/json" \
     -H "authorization: Basic YWRtaW46YWRtaW4=" \
     -H "Content-Type: application/json" \ 
     -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"force\": true}"
```

API endpoint used in this step: [UnregisterKubernetesCluster](ref:unregisterkubernetescluster)

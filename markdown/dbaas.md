---
slug: 'dbaas'
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
- https://percona-pmm.readme.io/reference/changesettings
- https://percona-pmm.readme.io/reference/registerkubernetescluster
- https://percona-pmm.readme.io/reference/createpxccluster
- https://percona-pmm.readme.io/reference/createpsmdbcluster

In this example we would use minikube for the kubernetes cluster and create PXC DB cluster, but similar API endpoints exist for the PSMDB.

### Enabling

To enable DBaaS, first we need:
- Enable DBaaS in settings.
- Specify the DNS name or public IP address of the pmm-server instance to be able to monitor DB clusters we create in DBaaS and Kubernetes cluster itself.

It is highly recommended to **use DNS name** instead of IP address but in example bellow we have a dev environment and use IP address instead.
#### Get Docker container IP and set it as public address

First of all we should get IP address of PMM (or DNS name should be used and that is recommended). If you are running in local minikube environment you can use following command to get IP address:
```bash
IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' pmm-server)
```
If your kubernetes cluster is located outside of your local system you can get public address by calling `ifconfig`.

Then to enable DBaaS send request to `Settings/Change` endpoint like below where `IP` is public IP address or DNS name of PMM Server instance.
```bash
curl -X POST "http://localhost/v1/Settings/Change" -H "accept: application/json" -u "admin:admin" -H "Content-Type: application/json" -d "{ \"pmm_public_address\": \"${IP}\", \"enable_dbaas\": true}"
```

API endpoint used in this step: [Change settings](https://percona-pmm.readme.io/reference/changesettings).

### Registering new Kubernetes cluster

Once kubernetes cluster is created it should be registered in PMM where `my_cluster` is a name of kubernetes cluster which will be used later. `sed` command is used to remove newlines, otherwise this script doesn’t work.
```bash
KUBECONFIG=$(kubectl -- config view --flatten --minify | sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/\\n/g')

curl -X POST "http://localhost/v1/management/DBaaS/Kubernetes/Register" -H "accept: application/json" -u "admin:admin" -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"kube_auth\": { \"kubeconfig\": \"${KUBECONFIG}\" }}"
```
This command will register kubernetes cluster, start monitoring of kubernetes cluster and install required kubernetes operators.

API endpoint used in this step: [RegisterKubernetesCluster](https://percona-pmm.readme.io/reference/registerkubernetescluster)

### Create PXC Cluster

Once we registered kubernetes cluster we can use it’s name to create DB Clusters. Here is an example for PXC Cluster.

```bash
curl -X POST "http://localhost/v1/management/DBaaS/PXCCluster/Create" -H "accept: application/json" -u “admin:admin” -H "Content-Type: application/json" -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\", \"expose\": true, \"params\": { \"cluster_size\": 3, \"pxc\": { \"compute_resources\": { \"cpu_m\": 1000, \"memory_bytes\": 2000000000 }, \"disk_size\": 25000000000, \"image\": \"percona/percona-xtradb-cluster:8.0.25-15.1\" }, \"haproxy\": { \"compute_resources\": { \"cpu_m\": 1000, \"memory_bytes\": 2000000000 } } }}"
```

API endpoint used in this step: [CreatePXCCluster](https://percona-pmm.readme.io/reference/createpxccluster).

### List Kubernetes clusters

Once you created PXC cluster you can check the status of the cluster by calling `List` endpoint.
```bash
curl -X POST "http://localhost/v1/management/DBaaS/DBClusters/List" -H "accept: application/json" -u “admin:admin” -H "Content-Type: application/json" -d "{ \"kubernetes_cluster_name\": \"my_cluster\"}"
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

API endpoint used in this step: [ListDBClusters](https://percona-pmm.readme.io/reference/listdbclusters)

### Get credentials

Once PXC Cluster is ready we can request credentials to connect to DB.

```bash
curl -X POST "http://localhost/v1/management/DBaaS/PXCClusters/GetCredentials" -H "accept: application/json" -u “admin:admin” -H "Content-Type: application/json" -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\"}"
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

API endpoint used in this step: [GetPXCClusterCredentials](https://percona-pmm.readme.io/reference/getpxcclustercredentials)

### Delete DB Cluster

If we don’t need DB Cluster anymore we can delete it using request below.
```bash
curl -X POST "http://localhost/v1/management/DBaaS/DBClusters/Delete" -H "accept: application/json" -u “admin:admin” -H "Content-Type: application/json" -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"name\": \"my-cluster-1\", \"cluster_type\": \"DB_CLUSTER_TYPE_PXC\"}"
```

API endpoint used in this step: [DeleteDBCluster deletes](https://percona-pmm.readme.io/reference/deletedbcluster)

### Unregister Kubernetes Cluster

After we played with DBaaS we can unregister kubernetes cluster. 

Unregister a kubernetes cluster doesn’t delete anything, it just removes the cluster from the list of registered clusters and all database clusters will remain active and will send metrics to PMM.

```bash
curl -X POST "http://localhost/v1/management/DBaaS/Kubernetes/Unregister" -H "accept: application/json" –u “admin:admin" -H "Content-Type: application/json" -d "{ \"kubernetes_cluster_name\": \"my_cluster\", \"force\": true}"
```

API endpoint used in this step: [UnregisterKubernetesCluster](https://percona-pmm.readme.io/reference/unregisterkubernetescluster)

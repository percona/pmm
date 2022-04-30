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

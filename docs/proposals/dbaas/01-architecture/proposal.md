# Kubernetes-native implementation for PMM/DBaaS

## Summary

PMM DBaaS has a lot of functionality to install/manage operators and CRUD operations for either a Percona XtraDB or a Percona Server For MongoDB cluster. However, the current implementation is not extensible enough and lacks scalability features.

This proposal introduces architectural changes that are required to make DBaaS more extensible and more native for K8S.

## Motivation

The current architecture for DBaaS has the following components:

1. PMM that exposes REST API for UI and some automation
2. DBaaS-controller exposes gRPC API to the PMM and works with Kubernetes cluster

![Current DBaaS architecture](./current_arch.jpg)

As a proof of concept, this architecture covers everything. However, it has the following issues

1. Non-native Kubernetes API to work with clusters. DBaaS controller has only gRPC API and this creates additional friction for community users to work/extend with the controller. One needs to generate a gRPC client to communicate with the DBaaS controller. Thus implementing the integration testing framework becomes a complex task to solve because popular frameworks such as codecept.js or playwright do not have gRPC support.
2. DBaaS controller has a set of CRUD endpoints for each database type (e.g. PXC and PSMDB). It adds additional room for bugs/inconsistencies and has the following issues
    * There’s no simplified and generic API for any database cluster
    * PMM needs to make two requests to get a list of created clusters (one for PSMDB and one for PXC clusters). In case of adding new database support, the DBaaS controller should have an additional set of CRUD endpoints and PMM should also call the list method for the new database type.

3. DBaaS controller has a lack of test coverage and integration testing because of the reasons above. Yet we can create an integration testing framework and increase coverage but in that case, it’ll cost a lot of time.
4. Currently, the DBaaS controller has only basic features such as CRUD operators for the database cluster and a lack of backup/restore features/additional configuration. There’s no way to specify additional parameters (Database configuration options, load balancer rules, storage class, backup schedule)
5. REST API for PMM does not follow REST guidelines.

Moving to OLM and a DBaaS operator will improve this situation.

## Goals

1. Make DBaaS more Kubernetes native and so make it the first-class citizen in the Kubernetes ecosystem.
2. Improve the overall quality of DBaaS by adding an integration testing framework
3. Improve performance of PMM/DBaaS feature by using native ways of communication with Kubernetes. PMM will directly call k8s API endpoints and use client-go caches for large-scale deployments.
4. Reduce the complexity of installing/managing operators in terms of updating/upgrading operators when there’s a new release.
5. Provide generic specifications to create/edit/delete a database cluster.
6. Provide generic specifications to backup/restore a database cluster inside Kubernetes.
7. Provide REST API that follows guidelines and provides a better developer experience for the automation and integration with PMM/DBaaS.
8. Provide a simplified way to create templates for a database cluster creation with pre-filled defaults.

## Non-goals

1. Bloat size of PMM docker container
2. Overcomplicate architecture
3. Overcomplicate the overall development process


## Proposal

The main idea is to move from the dbaas-controller that manages operators’ installation/updating its version database creation/management inside Kubernetes to a `dbaas-operator` and [OLM](https://olm.operatorframework.io/). The high-level architecture is described below.

![Proposed architecture](./proposed_arch_full.jpg)

In that case, the `dbaas-operator` provides Kubernetes native implementation of the creating/managing database clusters and acts like `unified operator to create any database cluster`. OLM installs and updates a version of the operators (psmdb, pxc, dbaas, victoria metrics).

### Working with the operators

An Operator catalog is a repository of metadata that Operator Lifecycle Manager (OLM) can query to discover and install Operators and their dependencies on a cluster. OLM always installs Operators from the latest version of a catalog. We can store Percona operators as well as Community operators inside Percona Catalog as shown below.

![DBaaS Catalog architecture](./dbaas_catalog.jpg)

Catalog also gives us the benefits of using CatalogSource and ClusterServiceVersion. ClusterServiceVersion (CSV) can solve the issue by getting the current version of installed operators via OLM API. Currently, we rely on the API version of Percona Operators by requesting `api-versions` via `kubectl` and using the latest API version as an operator’s version. However, there'll be no way to get information about the version after the 1.13 release of PSMDB and the 1.12 release of PXC operator. In that case, we can request an operator image and split the version information from the tag, but in the case of the overridden image tag, we can have discrepancies between versions. With OLM we’ll have a single source of truth for the installed operator version. [Read more](https://olm.operatorframework.io/docs/concepts/crds/clusterserviceversion/)

Example for PXC operator (It's only 1.10 available on operatorhub, however it demonstrates the idea)

```bash
kubectl get csv -n my-percona-xtradb-cluster-operator
NAME                                      DISPLAY                                   VERSION   REPLACES                                 PHASE
percona-xtradb-cluster-operator.v1.10.0   Percona Distribution for MySQL Operator   1.10.0    percona-xtradb-cluster-operator.v1.9.0   Installing
```

In that case, PMM/DBaaS will have a consistent way to get operator version. In addition, OLM can manage a complex upgrades for operatos via [InstallPlan](https://olm.operatorframework.io/docs/concepts/crds/installplan/), [OperatorGroup](https://olm.operatorframework.io/docs/concepts/crds/operatorgroup/) or [UpgradeGraph](https://olm.operatorframework.io/docs/glossary/#update-graph).

A high-level architecture diagram of OLM
![OLM Architecture](./olm_arch.jpg)

High-level sequence diagram that illustrates an Installation and upgrading process inside PMM/DBaaS in terms of end user

```mermaid
sequenceDiagram
    autonumber
    PMM UI->>ManageD: Add kubernetes cluster
    ManageD->>K8S: Install OLM
    break when olm is installed 
    	K8S->>OLM: provisions an OLM
    end
    Note right of K8S: OLM is provisioned!
    ManageD->>OLM: Install required operators and create a manual subscription 
    OLM->>Catalog: Gets the latest version of the operator 
    loop when subscription is created 
        OLM->>Catalog: manages subscription and checks for a new version
    end
    Note right of OLM: A new version of operator is available
    OLM->>Catalog: Downloads and installs a new version of operator and waiting for a user approval to switch to a new version.
    PMM UI->>ManageD: Check for operator update
    ManageD->>OLM: Is there an update available?
    OLM->>ManageD: Sure. I downloaded it for you and waiting for your approval to switch to a new version
    ManageD->>PMM UI: Yeah, there's a new version of operator available
    Note left of PMM UI: A user approved the upgrade request
    PMM UI->> ManageD: approve a new version of operator
    ManageD->>OLM: approve an subscription upgrade request
    Note left of OLM: A new version of operator is available 
```    
This process makes version service usage for supported versions of operators obsolete because it gets the latest version of available operator from the catalog. However, the catalog can be managed by our team and we'll push only supported versions of operators to the catalog. This also will improve integration testing for unreleased versions of operators.

### Working with the databases

The second part of the proposal is implementing the generic and simplified API to create database clusters via k8s operator. The proposal discusses the drawbacks of the current implementation in the Motivation section and moving to the `dbaas-operator` will have the following benefits 

1. Kubernetes native way to work with database clusters via unified and general specifications.
2. PMM/DBaaS does need to care about the state of a managed database.
3. PMM/DBaaS should manage clusters that were created via PMM/DBaaS.Yet,  it manages clusters that were created via Percona operators now. `dbaas-operator` can use its own kubernetes annotations to specify database clusters managed by PMM/DBaaS. Also, it opens room for additional features such as CR templates to create a database cluster and update a version of a template using the same annotations spec.
4. Installation and upgrading of `dbaas-operator` will be managed by OLM as proposed above.
5. `dbaas-operator` opens a possibility to create an easy to use integration testing framework using codecept.js/playwright or even simple bash scripts. Integration testing can be covered by development team. It can use Github actions as a pipeline to run additional tests to check create/load/run queries against exposed/non-exposed database clusters. 
6. PMM/DBaaS will have native integration with `dbaas-operator` via `client-go` package that will improve the overall performance on large-scale clusters 

The sequential diagram illustrates a creation of PXC cluster

```mermaid
sequenceDiagram
    autonumber
    PMM UI->>ManageD: Create a Database cluster
    Note right of PMM UI: (DatabaseType: PXC)
    ManageD->>K8S: Create a Database Kind
    K8S->>DBaaS Operator: Create a Database Kind object
    DBaaS Operator->>PXC Operator: Create a PXC cluster
    Note right of DBaaS Operator: Binds objects to each other
    Note right of DBaaS Operator:  (Database->PerconaXtraDBCluster)
    break when PXC cluster is in ready state
    	PXC Operator->>Statefulset: provisions PXC cluster
    end
    Note right of PXC Operator: PXC Cluster is created
    PMM UI->>ManageD: Get me a list of database clusters 
    ManageD->>K8S: Get list of Database clusters 
    K8S->>DBaaS Operator: Get list Database clusters
    DBaaS Operator->>K8S: returns array of Database objects
    Note right of K8S: They can be with DatabaseType either PXC or PSMDB
    K8S->>ManageD: Returns list of databases
    ManageD->>PMM UI: Returns list of database clusters
    Note right of PMM UI:  Pagination is available by default as well as caches
    
```    

#### Communication speed benchmarks

Currently [dbaas-controller](https://github.com/percona-platform/dbaas-controller) implements proxy api to k8s Percona operators thus `dbaas-controller` needs to call k8s and gets data from k8s operators. That could be expensive call to k8s as api returns full spec and in case of 100 or 1000 of server it could be quite large and time consuming response.

Also if there is no good connection to the k8s cluster situation would be much worse.

Here are some measurements made against PMM in different setups (local PMM -> Remote k8s, Remote PMM inside the same cluster, Direct API calls against K8S)

**PMM inside the same k8s cluster**

```
TIME="%e\n" time curl -k -s -o file.out --request POST --url https://172.105.146.244/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46VyUrMS5bVCszMV5LemQ7Nw==' --header 'content-type: application
/json' --data '
{
     "kubernetes_cluster_name": "default-pmm-cluster"
}
'
7.42

TIME="%e\n" time curl -k -s -o file.out --request POST --url https://172.105.146.244/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46VyUrMS5bVCszMV5LemQ7Nw==' --header 'content-type: application/json' --data '
{
     "kubernetes_cluster_name": "default-pmm-cluster"
}
'
7.90
```

**PMM runs on local machine and communicates with remote k8s cluster**

```
TIME="%e\n" time curl -k -s -o file1.out --request POST --url https://localhost:8443/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46YWRtaW4=' --header 'content-type: application/json' --data '
{
     "kubernetes_cluster_name": "lke75856"
}
'

TIME="%e\n" time curl -k -s -o file1.out --request POST --url https://localhost:8443/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46YWRtaW4=' --header 'content-type: application/json' --data '
{
     "kubernetes_cluster_name": "lke75856"
}
'
30.66

TIME="%e\n" time curl -k -s -o file1.out --request POST --url https://localhost:8443/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46YWRtaW4=' --header 'content-type: application/json' --data '
{
     "kubernetes_cluster_name": "lke75856"
}
'
26.60

TIME="%e\n" time curl -k -s --request POST --url https://localhost:8443/v1/management/DBaaS/DBClusters/List --header 'accept: application/json' --header 'authorization: Basic YWRtaW46YWRtaW4=' --header 'content-type: application/json' --data '
{
     "kubernetes_cluster_name": "lke75856"
}
'
<html>
<head><title>504 Gateway Time-out</title></head>
<body>
<center><h1>504 Gateway Time-out</h1></center>
<hr><center>nginx</center>
</body>
</html>
60.04
```

**Listing database clusters that were created with crossplane (uses operator pattern).**

```
TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims?limit=5 | jq '.items | length'
0.47
5

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims?limit=5 | jq '.items | length'
0.32
5

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims | jq '.items | length'
1.46
20

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims | jq '.items | length'
0.56
20

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims | jq '.items | length'
0.57
20

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/dbaas.percona.com/v1alpha1/dbclaims | jq '.items | length'
0.53
20

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/pxc.percona.com/v1/perconaxtradbclusters | jq '.items | length'
0.54
10

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/pxc.percona.com/v1/perconaxtradbclusters | jq '.items | length'
1.09
10

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/pxc.percona.com/v1/perconaxtradbclusters | jq '.items | length'
2.03
10

TIME="%e\n" time curl -s -k --header "Authorization: Bearer $JWT_TOKEN_KUBESYSTEM_DEFAULT" $KUBE_API/apis/pxc.percona.com/v1/perconaxtradbclusters | jq '.items | length'
1.36
10
```

So, requesting a list of database clusters from `dbaas-operator` (it uses the same framework as well as crossplane) will improve request time drastically. In addition, PMM/DBaaS can use [client-go caches](https://pkg.go.dev/k8s.io/client-go/tools/cache) as well as [Limit and Continue](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions) params of metav1.ListOptions struct.


### User Stories

As an SRE person, I should be able to register the Kubernetes cluster using a service account without admin access to the cluster.

As an SRE person, I should be able to understand what’s going wrong during the bootstrapping DBaaS feature inside of PMM in case of insufficient permissions so that I can debug and solve my issues. (E.g. No permissions to run kube-state-metrics, pxc, or psmdb operator).

As an SRE person, I should be able to rename a Kubernetes cluster once it was provisioned automatically so that I can keep my naming conventions.

As an SRE person, I should be able to specify which database operators I need to install in the cluster.

As an SRE person, I should be able to create logical spaces to deploy databases so that I can easily split my environments. (e.g. dev namespace goes to the dev environment and the staging namespace goes to the staging environment. For the production environment I should be able to register and setup an additional cluster.)

As an SRE person, I should be able to limit access to create/edit/destroy database clusters for specified users so that no devs are bugging me to do it for them.

As an SRE person, I should be able to create a resource template for a database engine so that I don’t need to manually provide it every time.

As an SRE person, I should be able to create an engine configuration template for a database engine.

As an SRE person, I should be able to manage database engine versions that are allowed to use because I need to control which versions are used in my environment.

As an SRE person, I should be able to specify a backup schedule template for a database cluster.

As an SRE person, I should be able to configure storage for backups.

As a user, I should be able to deploy a database with the recommended defaults

As a user, I should be able to deploy a database with the selected version or cluster size

As a user, I should be able to select a resource template to deploy a database

As a user, I should be able to select a resource template and database engine configuration template to deploy a database

As a user, I should be able to edit a database cluster (If I have sufficient permissions)

As a user, I should be able to delete a database cluster (If I have sufficient permissions)

As a user, I should be able to create a database cluster from a provided backup file.

As a DBA, I should be able to tune performance for a database cluster.

As a DBA(?), I should be able to view cluster resources available before creating a database.


## Design Details

### PMM REST API high-level design 

```
POST /dbaas/kubernetes_clusters - Add k8s cluster to the PMM. Provision DBaaS feature. Kubeconfig should be base64 encoded. Extend with the namespace?
GET /dbaas/kubernetes_clusters - List all kubernetes clusters
GET /dbaas/kubernetes_cluster/{id} - get information about the selected k8s cluster
GET /dbaas/dbclusters - list of created database clusters
PUT /dbaas/dbclusters/{id} - update/edit a database cluster. 
GET /dbaas/dbclusters/{id} - get the detailed information about the database cluster
POST /dbaas/dbclusters - create a new database cluster. The database struct is described below
GET /dbaas/dbclusters/{id}/credentials - get credentials for the selected db cluster (it can be embedded to the LIstDBClusters endpoint)
GET /dbaas/templates - get list of created templates that can be used as a CR template on a database cluster creating 
POST /dbaas/templates - create a database cluster template
PUT /dbaas/templates/{id} - update a database cluster template
GET /dbaas/templates/{id} - get a database cluster template
GET /dbaas/backup_templates - get list of created templates for backup schedule 
POST /dbaas/backup_templates - create a backup schedule template
PUT /dbaas/backup_templates/{id} - update a backup schedule template
GET /dbaas/backup_templates/{id} - get a database schedule template
GET /dbaas/versions - get supported versions of database clusters 
GET /dbaas/operator-versions - get operators' versions information from OLM and
PUT /dbaas/operator-versions - upgrade operator via OLM
```
Go struct that represents the payload to create/list database clusters 
```go
type (
	// EngineType stands for the supported database engines. Right now it's only pxc
	// and psmdb. However, it can be ps, pg and any other source
	EngineType string
	// LoadBalancerType contains supported loadbalancers. It can be proxysql or haproxy
	// for PXC clusters and mongos for PSMDB clusters.
	//
	// Once PG support will be added, it can be pg-bouncer or something else.
	LoadBalancerType string
	// Database struct contains an unified API to create any database cluster
	// via PMM/DBaaS
	Database struct {
		// Database type stands for supported databases by the PMM API
		// Now it's pxc or psmdb types but we can extend it
		Database EngineType `json:"database_type"`
		// DatabaseVersion sets from version service and uses the recommended version
		// by default
		DatabaseVersion string `json:"databaseVersion"`
		// DatabaseConfig contains a config settings for the specified database
		DatabaseConfig string `json:"databaseConfig"`
		// ClusterSize is amount of nodes that required for the cluster.
		// A database starts in cluster mode if clusterSize >= 3.
		// There's a possibility to setup a single node cluster but it's unsafe operation
		// and does not support upgrades from 1 to 3 nodes.
		// However, for clusterSize >= 3 dbaas-operator can scale it up to 5-7-N nodes
		// and scale down to 3.
		ClusterSize int32 `json:"clusterSize"`
		// LoadBalancer contains a load balancer settings. For PXC it's haproxy
		// or proxysql. For PSMDB it's mongos.
		LoadBalancer struct {
			Type      LoadBalancerType            `json:"type,omitempty"`
			Resources corev1.ResourceRequirements `json:"resources,omitempty"`
		} `json:"loadBalancer"`
		Monitoring struct {
			// Agent contains an image path for the agent used for monitoring
			// It'll be a percona/pmm-client:2 by default
			Agent string `json:"agent,omitempty"`
			// PMM contains settings to setup integration with the PMM server.
			// It contains public address, login and password for the authentication
			// and resources to specify request/limit resources.
			PMM struct {
				PublicAddress string                      `json:"publicAddress,omitempty"`
				Login         string                      `json:"login,omitempty"`
				Password      string                      `json:"login,omitempty"`
				Resources     corev1.ResourceRequirements `json:"resources,omitempty"`
			} `json:"pmm,omitempty"`
			// Prometheus contains a configuration options to use prometheus
			// as a monitoring solution in case of using dbaas-operator
			// without PMM installation.
			// But it still requires a manual installation of operators via OLM or
			// by managing
			Prometheus struct {
				Resources corev1.ResourceRequirements `json:"resources,omitempty"`
			} `json:"prometheus,omitempty"`
		} `json:"monitoring"`
		DBInstanceSpec struct {
			CPU      string `json:"cpu,omitempty"`
			Memory   string `json:"cpu,omitempty"`
			DiskSize string `json:"diskSize,omitempty"`
		} `json:"dbInstance"`
		Backup struct {
			// TDB
		} `json:"backup,omitempty"`
	}

)

const (
	PXCEngine   EngineType = "pxc"
	PSMDBEngine EngineType = "psmdb"
)

```
Since, the main focus for the PMM/DBaaS is to support AWS EKS clusters. The main focus is supporting managed kubernetes clusters as a main provider. However, the requirements can be changed and PMM/DBaaS may support bare-metal or EC2 setups. That adds additional constraints on the architecture. Everything related to the current implementation of k8s provider is located under `managed/services/dbaas` folder. However, it would be great to have `managed/services/dbaas/provider.go` file with the following interface

```go
// Provider is the interface that a developer should implement to add support of any other provider that required for DBaaS.
// Currently, PMM supports only K8s/EKS provider but bare-metal setup or using EC2 instances support can be adopted by implementing this inteface
type Provider interface {
  // ProvisionCluster provisions a specified cluster. In this case, it'll install
  // all required operators that we need (pxc, psmdb, dbaas-operator and victoria metrics
  // operator via OLM
  ProvisionCluster() error
  // CleanupCluster cleans up cluster and removes VM operator
  // and, or, dbaas-operator, pxc operator and psmdb operator via --force tag.
  // At the moment PMM should remove VM operator only and keep everything that related to databases
  // untouched and the end user can decide what to clean
  CleanupCluster() error
  // CreateDatabase cluster creates a cluster by using Database struct
  // and it'll make a request to dbaas-operator to create a database
  CreateDatabase(Database) error
  // EditDatabase edit's deployed database CR specs by using Database struct
  EditDatabase(Database) error
  // UpgradeDatabase upgrades database to a desired version
  UpgradeDatabase(Database) error
  // DestroyDatabase destroys a database from the cluster
  DestroyDatabase(Database) error
  // UpdateClusterDependencies upgrades cluster dependencies. For k8s it upgrades dbaas and database operators versions and CR configuration
  UpdateClusterDependencies() error
}
```
Every additional provider should implement this interface. Kubernetes related implementations

1. `managed/services/dbaas/kubernetes` is a parent package that implements generic `Provider`
2. `managed/services/dbaas/kubernetes/k8sclient` contains the implementation using `client-go` to communicate with k8s cluster
3. `managed/services/dbaas/kubernetes/olmclient` contains the implementation for `OLM`
4. `managed/services/management/dbaas/` contains the REST API implementation for PMM UI and uses `managed/services/dbaas` package.

### Templates

There are number of possible template use cases:
- as a DBA I would like to template Cluster and DB parameters for number of different environments
- as a SRE I would like to template k8s and cloud infrastructural parameters

When DBA would like to have more control over DBs and DB Clusters (versions, sizing, backups, configs, parameters), SRE would like to control over Kubernetes and Cloud infrastructure (networking, storage, secrets) and how that infrastructure connected to applications (affinity, networking, rbac).

Percona Kubernetes operators have CRs that include both parts of those requirements:
- configs, sizing, backup parameters, cluster parameters (sharding, replsets)
- toleration, affinity, annotations, storage classes, networking

All those template requirements are subset of Kubernetes CR objects that are supported by operators and kubernetes addons (such as special annotations).

To define similar properties for different environments, clusters, namespaces - templates could also have different visibility:
- namespace
- environment
- cluster
- set of clusters

And that visibility would probably define where those will be stored:
- template for a namespace (prod/staging namespaces): as object in a namespace
- environment templates: template with environment labels to select from
- cluster wide templates: as object in a cluster for all namespaces
- global templates: in a external storage in CI/CD, or in PMM as app that manages many clusters

#### DatabaseCluster templates

`dbaas-operator` would provide simplifier interface to abstract internal mechanics of kubernetes and different clouds, so templates for it's interface are probably more related to Developer's and DBA's use case. And that makes `dbaas-operator` a more simpler interface that is just a subset of bigger CR template.

As these kind of templates are more higher level, they would take priority over more low level templates and could be defined later when more business logic on operating templates would be defined.

![Template Layers](./tpl_layers.png)

Proposal on a current stage is to propagate low level templates (Database Engine defined by SRE/DBA in a cluster) to a higher level for the know fields. When middle level and high level templates (DBA, Dev) business logic would be better known, it will just a simple layer on top low level template and be presented to the user instead.

TBD.

#### Database Engine Templates

These templates provide possibility to template full Database operator CR even for the fields that `dbaas-operator` don't really care or even don't know (annotations, parameters for the future CSI drivers and etc).

1. SRE/DBA creates DB templates (through UI/API/YAML) and they are stored in the Kubernetes cluster.
2. UI requests [PMM API](proposal.md#pmm-rest-api-high-level-design) `GET /dbaas/templates`
3. PMM gets templates from k8s cluster
4. User chooses template in UI.
5. UI parses and present fields from template that are recognized
6. User creates database with or without modifications to the values from template
7. UI call PMM with results of user choices `POST /dbaas/dbclusters`

Priority:
  1. changes from PMM UI configured by user
  2. chosen template
  3. defaults from `dbaas-operator`

`dbaas-operator` merges all those 3 sources according to priority and produces CR (custom resource) for specific engine.

Corner cases:
- user uses PMM UI and there are no default template (or none selected)
  - PMM uses old scheme with version service and some default values
- user uses k8s directly without PMM and there neither default templates and submits nor optional fields 
  - `dbaas-operator` uses default CR that comes from alm-examples of CSV

There are might be couple of ways for `dbaas-operator` to know about templates.

Also this use case defaults to namespace and cluster visibility. So if there would be a need to have similar template for different clusters/namespaces - SRE need to copy them over to the another cluster/namespace and apply there (or use automation for it).

**New parameter in API**
```go
          // Optional unique template name for specified EngineType
		TemplateName string `json:"templateName,omitempty"`
```
`dbass-operator` receives `TemplateName` parameters and get template from k8s by combining engine type and template name. Or would implement additional controller for templates and track them.

That is a possible way is to extend `dbaas-operator` with template functionality. But there are too many questions open:
- what type to choose (string, unstructured)
- how to validate that parameters in template are correct
- how to track template versioning if it will be a controller
- how to extend templates when adding new Database operators

So instead of committing to the API, annotations could solve the problem for now.

**Annotations**

It could be a good solution to not blow up API and keep it simple by making that convention with annotations and later decide to keep it or to switch to something else. Annotations are more flexible and are conventions instead of hard contract, thus we could avoid migrations/mutations later.

Annotations that PMM or user could set for `dbaas-operator` that related to the templates:
- `dbaas.percona.com/dbtemplate-kind: PSMDBtemplate`: is CustomResource (CR) Kind that implements template
- `dbaas.percona.com/dbtemplate-name: prod-app-X-small`: `metadata.name` identifier for CR that provides template
- `dbaas.percona.com/origin: pmm`: who created CR for `database-operator`: pmm, user, sre, dba, ci

If one of those 2 parameters (kind, name) are not set - `dbaas-operator` wouldn't be able to identify template and thus would ignore another one.

Annotations that are set by `dbaas-operator` for any engine to provide additional information about inputs for template:
- all annotations from the template should be propagated to CR of the DB Cluster CR.
- `dbaas.percona.com/dbtemplate-kind: PSMDBtemplate`
- `dbaas.percona.com/dbtemplate-name: prod-app-X-small`
- `dbaas.percona.com/origin: pmm`
- `dbaas.percona.com/dbtemplate-default: yes`: default CR was created without origin and parameters
- more annotations are TBD

**Labels**

All the labels of the DB Template should be propagated to the DB Cluster CR.

**Database Engine Templates implementation**

On a first phase to simplify template creation it might be better to restrict templates to simple k8s custom objects without operator that handles them. Architecture for interacting and layering for some additional operators is not yet defined.

To simplify implementation of `dbaas-operator` it is better to build templates with exactly the same definitions as the corresponding CR of DB Cluster instances. So `dbaas-operator` would just blindly merge template with user input without validating and knowing details of template implementation. Thus templates should have exactly same CRD as CRD for DB clusters. CRD for templates are needed to avoid errors as templates would be validated by k8s.

Here are CRDs for DB Clusters:
- PSMDBtemplate: [psmdbtpl.dbaas.percona.com.crd.yaml](psmdbtpl.dbaas.percona.com.crd.yaml)
- PXCtemplate: [pxctpl.dbaas.percona.com.crd.yaml](pxctpl.dbaas.percona.com.crd.yaml)

Example of CRs for templates:
- [psmdbtpl.dbaas.percona.com.cr.yaml](psmdbtpl.dbaas.percona.com.crd.yaml)
- [pxctpl.dbaas.percona.com.cr.yaml](pxctpl.dbaas.percona.com.crd.yaml)

CRs should have these annotations:
- `dbaas.percona.com/dbtemplate-origin: sre`: who created template: sre, dba and etc
- TBD

CRs should have these labels:
- `IsDefaultTpl: "yes"`: yes, or no
- `dbaas.percona.com/engine: psmdb`: engine type
- `dbaas.percona.com/template: "yes"`: indicates that this is a template
- TBD

There could be more labels to identify env, teams and defaults for teams. That logic could be defined later and probably defined in a separate document with versions of apps that could understand such convention.

Here is example of 2 templates creation:
```sh
minikube start

kubectl apply -f pxctmpl.dbaas.percona.com.crd.yaml 
customresourcedefinition.apiextensions.k8s.io/pxctemplates.dbaas.percona.com created

kubectl apply -f psmdbtpl.dbaas.percona.com.crd.yaml 
customresourcedefinition.apiextensions.k8s.io/psmdbtemplates.dbaas.percona.com created

kubectl apply -f pxctpl.dbaas.percona.com.cr.yaml 
pxctemplate.dbaas.percona.com/prod-app-n-large created

kubectl apply -f psmdbtpl.dbaas.percona.com.cr.yaml 
psmdbtemplate.dbaas.percona.com/dev-app-x created

kubectl get psmdbtpl,pxctpl
NAME                                        ENDPOINT   STATUS   AGE
psmdbtemplate.dbaas.percona.com/dev-app-x                       31s

NAME                                             ENDPOINT   STATUS   PXC   PROXYSQL   HAPROXY   AGE
pxctemplate.dbaas.percona.com/prod-app-n-large  

```

PMM could get templates by selecting specific dbaas resources:
```sh
kubectl api-resources --api-group=dbaas.percona.com
NAME             SHORTNAMES   APIVERSION                   NAMESPACED   KIND
psmdbtemplates   psmdbtpl     dbaas.percona.com/v1alpha1   true         PSMDBtemplate
pxctemplates     pxctpl       dbaas.percona.com/v1alpha1   true         PXCtemplate

kubectl api-resources --api-group=dbaas.percona.com --verbs=list --namespaced -o name | xargs -n 1 kubectl get -l 'dbaas.percona.com/engine in (psmdb,pxc)',dbaas.percona.com/template="yes"
NAME        ENDPOINT   STATUS   AGE
dev-app-x                       7s
NAME               ENDPOINT   STATUS   PXC   PROXYSQL   HAPROXY   AGE
prod-app-n-large                                                  5m24s
```

### Test Plan

During moving from dbaas-controller to `dbaas-operator` we'll keep the same user experience for the end user

#### Prerequisite testing updates
// TBD
#### Unit tests
// TBD
#### Integration tests
// TBD
#### e2e tests
// TBD
### Graduation Criteria
// TBD

## Production Readiness Review Questionnaire
### Feature Enablement and Rollback
// TBD
### Rollout, Upgrade and Rollback Planning
// TBD
### Dependencies

There will be a new dependencies added to the PMM
1. client-go
2. olmclient
3. dbaas-operator
4. 

### Scalability
// TBD


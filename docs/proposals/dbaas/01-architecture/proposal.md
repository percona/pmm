# Kubernetes-native implementation for DBaaS

## Summary

PMM/DBaaS feature has a lot of functionality to install/manage operators and CRUD operations for Percona XtraDB/Percona Server For MongoDB clusters. However, the current implementation is not extensible enough and lacks scalability features.

This proposal introduces architectural changes that are required to make DBaaS more extensible and more native for K8S

## Motivation

The current architecture for DBaaS has the following components

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

// To be defined

## Proposal

The main idea is to move from the dbaas-controller that manages operators’ installation/updating its version database creation/management inside Kubernetes to a dbaas-operator and [OLM](https://olm.operatorframework.io/). The high-level architecture is described below.

![Proposed architecture](./proposed_arch_full.jpg)

In that case, the dbaas-operator provides Kubernetes native implementation of the creating/managing database clusters and acts like `unified operator to create any database cluster`. OLM installs and updates a version of the operators (psmdb, pxc, dbaas, victoria metrics).

### Working with the operators 

### Working with the databases 

## User Stories (Optional)

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

## Notes/Constraints/Caveats (Optional)
## Risks and Mitigations

## Design Details

```go

package dbaas

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
	// Provider is the interface that a developer should implement to add support of any other provider that required for DBaaS.
	// Currently, PMM supports only K8s/EKS provider but bare-metal setup or using EC2 instances support can be adopted by implementing this inteface
	Provider interface {
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
)

const (
	PXCEngine   EngineType = "pxc"
	PSMDBEngine EngineType = "psmdb"
)

```

## Test Plan

During moving from dbaas-controller to dbaas-operator we'll keep the same user experience for the end user

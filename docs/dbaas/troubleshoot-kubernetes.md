## Troubleshooting Kubernetes provisioning

!!! caution alert alert-primary "Do not use for mission-critical workloads"
    DBaaS feature is deprecated. We encourage you to use [Percona Everest](http://per.co.na/pmm-to-everest) instead. Check our [Migration guide](http://per.co.na/pmm-to-everest-guide).


There are two things that might go wrong during the provisioning:

1. OLM installation
2. Operators installation

### Troubleshooting OLM installation

By default OLM installs its operators and catalogs into `olm` namespace. Here's the example of successful installation of OLM

```
kubectl get pods -n olm
NAME                                                              READY   STATUS      RESTARTS   AGE
catalog-operator-67bcbb4f5d-jmdhv                                 1/1     Running     0          16h
olm-operator-d9f76fdc9-7vdx2                                      1/1     Running     0          16h
operatorhubio-catalog-4gzfk                                       1/1     Running     0          74s
packageserver-877948445-swdq5                                     1/1     Running     0          16h
packageserver-877948445-sz44r                                     1/1     Running     0          16h
percona-dbaas-catalog-777hb                                       1/1     Running     0          16h
```

Note: the output may be longer but it shows what needs to be installed automatically

1. OLM Operator is responsible for deploying applications defined by CSV resources after the required resources specified in the CSV are present in the cluster.
2. The Catalog Operator is responsible for resolving and installing CSVs and the required resources they specify. It is also responsible for watching CatalogSources for updates to packages in channels and upgrading them (optionally automatically) to the latest available versions.
3. Packageserver is responsible for providing metadata to the operators like ClusterServiceVersion that is used for installing/upgrading operators.
4. percona-dbaas-catalog is Percona's managed catalog that defines which operators' versions are available. This component has information about tested and supported versions of operators.

You can use `kubectl describe pod -n olm podName` to understand what went wrong during the installation.

### Troubleshooting operators installation

Once OLM is installed, PMM does the following actions to install each operator:

1. Creates a subscription for an operator
2. Approves the first available install plan automatically

Once the install plan is approved OLM will create a corresponding ClusterServiceVersion automatically and install the operator.

During this process the following steps might go wrong

1. Subscription was not created
2. Install plan was not created
3. Installation of CSV failed

#### Listing subscriptions

```
 kubectl get sub
NAME                              PACKAGE                           SOURCE                  CHANNEL
dbaas-operator                    dbaas-operator                    percona-dbaas-catalog   stable-v0
percona-server-mongodb-operator   percona-server-mongodb-operator   percona-dbaas-catalog   stable-v1
percona-xtradb-cluster-operator   percona-xtradb-cluster-operator   percona-dbaas-catalog   stable-v1
victoriametrics-operator          victoriametrics-operator          percona-dbaas-catalog   stable-v0
```
Note: Names may may vary depending on the version of PMM you’re using

#### Listing install plans

Once subscriptions are created and the catalog operator found a CSV to be installed it creates install plans for each operator which are approved by PMM automatically during the provisioning process. The upgrading process should be approved by a user.

```
kubectl get ip
NAME            CSV                                       APPROVAL   APPROVED
install-6gp4k   percona-xtradb-cluster-operator.v1.11.0   Manual     true
install-77jtx   victoriametrics-operator.v0.29.1          Manual     true
install-7lgsj   victoriametrics-operator.v0.29.1          Manual     true
install-bjmvj   victoriametrics-operator.v0.29.1          Manual     true
```

#### Listing CSV or how to be sure that operators were installed

```
kubectl get csv
NAME                                      DISPLAY                                                      VERSION   REPLACES                                  PHASE
dbaas-operator.v0.0.19                    DBaaS operator                                               0.0.19                                              Succeeded
percona-server-mongodb-operator.v1.11.0   Percona Distribution for MongoDB Operator                    1.11.0    percona-server-mongodb-operator.v1.10.0   Succeeded
percona-xtradb-cluster-operator.v1.11.0   Percona Operator for MySQL based on Percona XtraDB Cluster   1.11.0    percona-xtradb-cluster-operator.v1.10.0   Succeeded
victoriametrics-operator.v0.29.1          VictoriaMetrics Operator                                     0.29.1    victoriametrics-operator.v0.27.2          Succeeded
```

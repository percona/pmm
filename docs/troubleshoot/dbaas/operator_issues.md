# Troubleshooting operators installation

Once OLM is installed, PMM does the following actions to install each operator:
{.power-number}

1. Creates a subscription for an operator
2. Approves the first available install plan automatically

Once the install plan is approved OLM will create a corresponding ClusterServiceVersion automatically and install the operator.

During this process the following steps might go wrong:
{.power-number}

1. Subscription was not created
2. Install plan was not created
3. Installation of CSV failed

## Listing subscriptions

```
 kubectl get sub
NAME                              PACKAGE                           SOURCE                  CHANNEL
dbaas-operator                    dbaas-operator                    percona-dbaas-catalog   stable-v0
percona-server-mongodb-operator   percona-server-mongodb-operator   percona-dbaas-catalog   stable-v1
percona-xtradb-cluster-operator   percona-xtradb-cluster-operator   percona-dbaas-catalog   stable-v1
victoriametrics-operator          victoriametrics-operator          percona-dbaas-catalog   stable-v0
```
Note: Names may may vary depending on the version of PMM you’re using

## Listing install plans

Once subscriptions are created and the catalog operator found a CSV to be installed it creates install plans for each operator which are approved by PMM automatically during the provisioning process. The upgrading process should be approved by a user.

```
kubectl get ip
NAME            CSV                                       APPROVAL   APPROVED
install-6gp4k   percona-xtradb-cluster-operator.v1.11.0   Manual     true
install-77jtx   victoriametrics-operator.v0.29.1          Manual     true
install-7lgsj   victoriametrics-operator.v0.29.1          Manual     true
install-bjmvj   victoriametrics-operator.v0.29.1          Manual     true
```

## Listing CSV or how to be sure that operators were installed

```
kubectl get csv
NAME                                      DISPLAY                                                      VERSION   REPLACES                                  PHASE
dbaas-operator.v0.0.19                    DBaaS operator                                               0.0.19                                              Succeeded
percona-server-mongodb-operator.v1.11.0   Percona Distribution for MongoDB Operator                    1.11.0    percona-server-mongodb-operator.v1.10.0   Succeeded
percona-xtradb-cluster-operator.v1.11.0   Percona Operator for MySQL based on Percona XtraDB Cluster   1.11.0    percona-xtradb-cluster-operator.v1.10.0   Succeeded
victoriametrics-operator.v0.29.1          VictoriaMetrics Operator                                     0.29.1    victoriametrics-operator.v0.27.2          Succeeded
```




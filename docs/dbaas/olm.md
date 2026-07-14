---
slug: dbaas
---

## How OLM (Operator Lifecycle Manager) works.

DBaaS leverages the installation and upgrade of operators on OLM.
You must create an operator group and a subscription to install an operator.
The official documentation with detailed examples can be found [here](https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/).

DBaaS installs the following operators by default:
- OLM 
- DBaaS
- PSMDB
- PXC

You can manually list the subscriptions using kubectl:
```
kubectl get subscriptions
NAME                                 PACKAGE                           SOURCE                  CHANNEL
my-percona-server-mongodb-operator   percona-server-mongodb-operator   operatorhubio-catalog   stable
my-percona-xtradb-cluster-operator   percona-xtradb-cluster-operator   operatorhubio-catalog   stable
```

### Known issue
When two or more operators are pending installation approval, OLM creates a second subscription that includes both operators. 
will have both operators. Listing the install plans could be confusing. For example:
```
kubectl get installplans
NAME            CSV                                       APPROVAL   APPROVED
install-9rxvz   percona-server-mongodb-operator.v1.13.1   Manual     false
install-mghbh   percona-server-mongodb-operator.v1.13.1   Manual     false
```
Although both install plans seem to be for PSMDB, it's worth examining each separately:

**First install plan**

```
kubectl get installplan install-9rxvz -oyaml
apiVersion: operators.coreos.com/v1alpha1
kind: InstallPlan
metadata:
  creationTimestamp: "2023-03-07T12:36:28Z"
  generateName: install-
  generation: 1
  labels:
    operators.coreos.com/percona-server-mongodb-operator.default: ""
  name: install-9rxvz
  namespace: default
  ownerReferences:
  - apiVersion: operators.coreos.com/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Subscription
    name: my-percona-server-mongodb-operator
    uid: 2581b852-36b3-41e3-92f0-02a4f2ebb05d
  resourceVersion: "1037"
  uid: d02807a7-3b24-49eb-b12b-63bc1ef817d6
spec:
  approval: Manual
  approved: false
  clusterServiceVersionNames:
  - percona-server-mongodb-operator.v1.13.1
  generation: 1
```

**Second install plan**

```
kubectl get installplan install-mghbh -oyaml
apiVersion: operators.coreos.com/v1alpha1
kind: InstallPlan
metadata:
  creationTimestamp: "2023-03-07T12:41:46Z"
  generateName: install-
  generation: 1
  labels:
    operators.coreos.com/percona-xtradb-cluster-operator.default: ""
  name: install-mghbh
  namespace: default
  ownerReferences:
  - apiVersion: operators.coreos.com/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Subscription
    name: my-percona-server-mongodb-operator
    uid: 2581b852-36b3-41e3-92f0-02a4f2ebb05d
  - apiVersion: operators.coreos.com/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Subscription
    name: my-percona-xtradb-cluster-operator
    uid: 6796d009-9a29-49b4-af9b-6af09a895317
  resourceVersion: "1314"
  uid: f6a87327-1f60-4cae-8b7a-d881c8c522c2
spec:
  approval: Manual
  approved: false
  clusterServiceVersionNames:
  - percona-server-mongodb-operator.v1.13.1
  - percona-xtradb-cluster-operator.v1.12.0
  generation: 2
```
Spec section of the second install plan indicates it will handle the installation of both operators:
```
spec:
  approval: Manual
  approved: false
  clusterServiceVersionNames:
  - percona-server-mongodb-operator.v1.13.1
  - percona-xtradb-cluster-operator.v1.12.0
```
We can only determine the operators being handled by an install plan if we receive the details in YAML or JSON format.```

## Conclusion
The short version `kubectl get installplans` will show only the first operator in the list and this can be confusing and misleading but it is not
a consequence of DBaaS.


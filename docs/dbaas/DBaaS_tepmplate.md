# Create a database cluster from a template

Database clusters can be created from templates using PMM. Templates allow you to customize the creation of database clusters. You can adapt templates to tweak K8s-specific settings such as **liveness probes**, changing **config maps**, or tuning **database engines**. 

## Create Custom Resource Definition (CRD) template

To create a template, do the following:

!!! note alert alert-primary "Note"   
    The example below shows how to change the `upgradeOptions` field, but it would be different if you wanted to customize some other field.

1. Identify the fields of interest by reading the operator documentation and corresponding CRDs. In this case, `updateStrategy` and `upgradeOptions` fields as per the [PXC operator documentation](https://docs.percona.com/percona-operator-for-mysql/pxc/update.html#manual-upgrade_1) and [PXC CRD](https://github.com/percona/percona-xtradb-cluster-operator/blob/v1.11.0/deploy/crd.yaml#L8379-L8392).

2. Create a template CRD `pxctpl-crd-upgrade-options.yaml` with the fields of interest as follows:

!!! note alert alert-primary "Note"   
    Template CRDs must have an `openAPIV3Schema` that must be a subset of the parent engine CRD. For this case, the parent engine CRD is [this](https://github.com/percona/percona-xtradb-cluster-operator/blob/v1.11.0/deploy/crd.yaml).    
    
```sh
apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    metadata:
    creationTimestamp: null
    name: pxctemplateupgradeoptions.dbaas.percona.com
    labels:
        dbaas.percona.com/template: "yes"
        dbaas.percona.com/engine: "pxc"
    spec:
    group: dbaas.percona.com
    names:
        kind: PXCTemplateUpgradeOptions
        listKind: PXCTemplateUpgradeOptionsList
        plural: pxctemplateupgradeoptions
        singular: pxctemplateupgradeoptions
    scope: Namespaced
    versions:
    - name: v1
        schema:
        openAPIV3Schema:
            properties:
            apiVersion:
                type: string
            kind:
                type: string
            metadata:
                type: object
            spec:
                properties:
                updateStrategy:
                    type: string
                upgradeOptions:
                    properties:
                    apply:
                        type: string
                    schedule:
                        type: string
                    versionServiceEndpoint:
                        type: string
                    type: object
                type: object
            status:
                type: object
            type: object
        served: true
        storage: true   
```

3. Run the following command:

    ```sh
    $ kubectl apply -f pxctpl-crd-upgrade-options.yaml
    ```
For more information, see [DatabaseCluster templates](https://github.com/percona/dbaas-operator/blob/main/docs/templates.md#creating-the-template-crd).

## Add Read permissions for pxctemplateupgradeoptions

For the dbaas-operator to apply the template it needs access to the template CRs.

```sh
$ DBAAS_OPERATOR_MANAGER_ROLE=$(kubectl get clusterroles | grep dbaas-operator | grep -v metrics | grep -v proxy | cut -f 1 -d ' '); kubectl get clusterroles/"$DBAAS_OPERATOR_MANAGER_ROLE" -o yaml > dbaas-operator-manager-role.yaml

$ cat <<EOF >>dbaas-operator-manager-role.yaml
- apiGroups:
  - dbaas.percona.com
  resources:
  - pxctemplateupgradeoptions
  verbs:
  - get
  - list
EOF
```

Run the following command to apply the configuration:

```sh
$ kubectl apply -f dbaas-operator-manager-role.yaml
```

## Create Custom Resources (CR) template

1. Create the CR `pxctpl-disable-automatic-upgrades.yaml` file with the desired values as follows:

    ```sh
    apiVersion: dbaas.percona.com/v1
    kind: PXCTemplateUpgradeOptions
    metadata:
    name: disable-automatic-upgrades
    labels:
        dbaas.percona.com/template: "yes"
        dbaas.percona.com/engine: "pxc"
    spec:
    updateStrategy: SmartUpdate
    upgradeOptions:
        apply: Disabled
    ```

2. Run the following command:

    ```sh
    $ kubectl apply -f pxctpl-disable-automatic-upgrades.yaml

    pxctemplateupgradeoptions.dbaas.percona.com/disable-automatic-upgrades created
    ```


## Create a DB cluster from template

To create a DB cluster from a template, do the following:

1. From the main menu navigate to <i class="uil uil-database"></i> *DBaaS* → *Create DB Cluster*.

2. On the *Advanced Settings* panel, select the template from the *Templates* drop-down.


    ![!](../_images/PMM_dbaas_template.png)


3. Click `Create`.







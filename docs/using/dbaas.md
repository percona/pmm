# DBaaS

!!! caution alert alert-warning "Caution"
    DBaaS functionality is currently in [technical preview](../details/glossary.md#technical-preview) and is subject to change.

The DBaaS dashboard is where you add, remove, and operate on Kubernetes and database clusters.

## Activate DBaaS

The DBaaS feature is turned off by default. To turn it on:

1. Go to <i class="uil uil-cog"></i> *Configuration* → <i class="uil uil-setting"></i> *Settings* → *Advanced Settings*.

2. Click the <i class="uil uil-toggle-off"></i> toggle in the *Technical preview features* section of the page.

## Open the DBaaS dashboard

From the left menu, select <i class="uil uil-database"></i> *DBaaS*.

![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Panel.png)



## Kubernetes clusters

## Add a Kubernetes cluster automatically

!!! note alert alert-primary "Availability"
    Automatic addition of k8s cluster in PMM is available starting with PMM 2.32.0.

If you deploy PMM in k8s, you must manually add these k8s to PMM. In order to simplify DBaaS, k8s clusters can now be automatically added to PMM as follows:

**Prerequisites:** Helm Chart/Manifest installed

1. Use [Helm](https://github.com/helm/helm) to install PMM server on Kubernetes clusters. For more information, see [documentation](../setting-up/server/helm.md).

  ```sh
  helm install pmm --set service.type="LoadBalancer" --set image.repository=percona/pmm-server --set image.tag=2.32.0 --set secret.pmm_password=admin percona/pmm
  ```

2. Log in to PMM.

3. [Enable PMM](dbaas.md#activate-dbaas) in DBaaS. 

4. You should see the default k8s cluster added in DBaaS.



### Add a Kubernetes cluster

!!! caution alert alert-warning "Caution"
    Ensure that you set PMM *Public Address* under <i class="uil uil-cog"></i> *Configuration* → <i class="uil uil-setting"></i> *Settings* → *Advanced Settings* before creating a Kubernetes cluster. Otherwise, PMM would **not** monitor the Kubernetes cluster along with the associated database clusters.


!!! note alert alert-primary ""
    PXC and PSMDB operators are installed as part of the Kubernetes cluster registration process. It enables you to deploy database clusters into the Kubernetes cluster.
    
    If a public address is set VM Operator is also installed as part of the Kubernetes cluster registration process. It lets you monitor a kubernetes cluster via PMM.

1. Click *Register new Kubernetes Cluster*.

2. Copy the value of *Kubeconfig file* and click *Paste from clipboard* to copy the content of the kubeconfig file in the corresponding field. The value of *Kubernetes Cluster Name* gets auto-populated from the contents of the *kubeconfig file*.

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.30.0.

!!! note alert alert-primary ""
    This feature is available only in [secure contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts) (HTTPS) and some [supporting browsers](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard#browser_compatibility).


![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Details.png)


!!! note alert alert-info ""
    For a Kubernetes cluster, when using Amazon Elastic Kubernetes Service (EKS) and the *kubeconfig file* does not contain the AWS 
    access key ID and AWS secret access key. Select the *Using Amazon Elastic Kubernetes Service (EKS)* 
    checkbox and enter the access key ID and secret access key in the respective fields.
    For information on obtaining these, see the [AWS documentation].



![!](../_images/PMM_DBaaS_Kubernetes_Cluster_EKS.png)

3. Click *Register*.

4. A message will momentarily display telling you whether the registration was successful or not.

    ![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Added.png)

### Unregister a Kubernetes cluster

!!! caution alert alert-warning "Important"
    You can't unregister a Kubernetes cluster if there DB clusters associated with it.

1. Click *Unregister*.

2. Confirm the action by clicking *Proceed*, or abandon by clicking *Cancel*.

### View a Kubernetes cluster's configuration

1. Find the row with the Kubernetes cluster you want to see.

2. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Show configuration*.

### Manage allowed component versions

Administrators can select allowed and default versions of components versions for each cluster.

1. Find the row with the Kubernetes cluster you want to manage.

2. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Manage versions*.

    ![!](../_images/PMM_DBaaS_Kubernetes_Manage_Versions.png)

3. Select an *Operator* and *Component* from the drop-down menus.

    ![!](../_images/PMM_DBaaS_Kubernetes_Manage_Components_Versions.png)

4. Activate or deactivate allowed versions, and select a default in the *Default* menu.

5. Click *Save*.

### Kubernetes operator status

The Kubernetes Cluster tab shows the status of operators.

![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Operator_Status.png)

### Kubernetes operator update

When a new version of the operator is available the *Operators* column shows a message with this information. Click the message to go to the operator release notes to find out more about the update.

![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Operator_Update.png)

To update the cluster:

1. Find the row with the operator you want to update.

2. Click the *Update* button in front of the operator.

3. Confirm the action by clicking *Update*, or abandon by clicking *Cancel*.

    ![!](../_images/PMM_DBaaS_Kubernetes_Cluster_Operator_Update_Confirmation.png)

## DB clusters

### Add a DB Cluster

You must create at least one Kubernetes cluster to create a DB cluster.

To monitor a DB cluster, set up a [public address](../how-to/configure.md#public-address) for PMM Server first.


####Add a one-click DB cluster


!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.30.0.

You can create a DB cluster literally at the click of a button. All the fields will be automatically populated with the default values. 

To create a DB cluster, do the following:

1. Select the *DB Cluster* tab.

2. Click *Create DB Cluster*.

3. Click *Create Cluster* to create your Cluster.

####Add a DB cluster with custom values

1. Select the *DB Cluster* tab.

2. Click *Create DB Cluster*.

3. In section 1, *Basic Options*:

    1. Enter a value for *Cluster name*. A cluster name:
        - must begin with a lowercase letter;
        - can comprise lowercase letters, numbers and dashes;
        - must end with an alphanumeric character.

    2. Select a cluster from the *Kubernetes Cluster* menu.

    3. Select a database type from the *Database Type* menu.

        ![!](../_images/PMM_DBaaS_DB_Cluster_Basic_Options_Filled.jpg)

4. Expand section 2, *Advanced Options*.

    1. Select *Topology*, either *Cluster* or *Single Node*.

    2. Select the number of nodes. (The lower limit is 3.)

    3. Select *External Access* if you want to make your DB cluster available outside of Kubernetes cluster.

        By default, only internal access is provided.

        *External Access* can't be granted for local Kubernetes clusters (e.g. minikube).

    4. Select a preset for *Resources per Node*.

        *Small*, *Medium* and *Large* are fixed preset values for *Memory*, *CPU*, and *Disk*.

        Values for the *Custom* preset can be edited.

        Beside each resource type is an estimate of the required and available resources represented numerically in absolute and percentage values, and graphically as a colored, segmented bar showing the projected ratio of used to available resources. A red warning triangle <i style="color: red" class="uil uil-exclamation-triangle"></i> is shown if the requested resources exceed those available.

        ![!](../_images/PMM_DBaaS_DB_Cluster_Advanced_Options.png)

5. When both *Basic Options* and *Advanced Options* section icons are green, the *Create Cluster* button becomes active. (If inactive, check the values for fields in sections whose icon is red.)

    Click *Create Cluster* to create your cluster.

6. A row appears with information on your cluster:

    ![!](../_images/PMM_DBaaS_DB_Cluster_Created.png)

    - *Name*: The cluster name.
    - *Database*: The cluster database type and version.
    - *Connection*:
        - *Host*: The hostname.
        - *Port*: The port number.
        - *Username*: The connection username.
        - *Password*: The connection password (click the eye icon <i class="uil uil-eye"></i> to reveal).
    - *DB Cluster Parameters*:
        - *K8s cluster name*: The Kubernetes cluster name.
        - *CPU*: The number of CPUs allocated to the cluster.
        - *Memory*: The amount of memory allocated to the cluster.
        - *Disk*: The amount of disk space allocated to the cluster.
    - *Cluster Status*:
        - *PENDING*: The cluster is being created.
        - *ACTIVE*: The cluster is active.
        - *FAILED*: The cluster could not be created.
        - *DELETING*: The cluster is being deleted.
        - *UPDATING*: The cluster is being updated.

### Delete a DB Cluster

1. Find the row with the database cluster you want to delete.

2. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Delete*.

3. Confirm the action by clicking *Proceed*, or abandon by clicking *Cancel*.

    ![!](../_images/PMM_DBaaS_DB_Cluster_Delete.png)

!!! danger alert alert-danger "Danger"
    Deleting a cluster in this way also deletes any attached volumes.

### Edit a DB Cluster

1. Select the *DB Cluster* tab.

2. Find the row with the database cluster you want to change.

3. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Edit*.

A paused cluster can't be edited.

### Restart a DB Cluster

1. Select the *DB Cluster* tab.

2. Identify the database cluster to be changed.

3. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Restart*.

### Suspend or resume a DB Cluster

1. Select the *DB Cluster* tab.

2. Identify the DB cluster to suspend or resume.

3. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click the required action:

    - For active clusters, click *Suspend*.

        ![!DBaaS Suspend](../_images/PMM_DBaaS_DB_Cluster_Suspend.gif)

    - For paused clusters, click *Resume*.

        ![!DBaaS Resume](../_images/PMM_DBaaS_DB_Cluster_Resume.gif)

### Update a DB Cluster

1. Select the *DB Cluster* tab.

2. Identify the DB cluster to update.

3. In the *Actions* column, open the <i class="uil uil-ellipsis-v"></i> menu and click *Update*:

    ![!DBaaS Update](../_images/PMM_DBaaS_DB_Cluster_Update_menu.png)

4. Confirm the update by clicking on *Update*, or abandon by clicking *Cancel*.

    ![!DBaaS Update Confirmation](../_images/PMM_DBaaS_DB_Cluster_Update_confirmation.png)

!!! seealso alert alert-info "See also"
    [Setting up a development environment for DBaaS](../setting-up/server/dbaas.md)

[ALPHA]: https://en.wikipedia.org/wiki/Software_release_life_cycle#Alpha
[Amazon Elastic Kubernetes Service (EKS)]: https://aws.amazon.com/eks/
[AWS documentation]: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html

## Install Operators on GKE

!!! alert alert-warning "Caution"
    These instructions are still in development.

**Prerequisites**

You should have an account on GCP [https://cloud.google.com/](https://cloud.google.com/).

1. Login into google cloud platform console [https://console.cloud.google.com/](https://console.cloud.google.com/)

2. Navigate to Menu --> Kubernetes Engine --> Clusters

    ![](../../_images/PMM_DBaaS_GKE_1.png)

3. Click button Create cluster

    ![](../../_images/PMM_DBaaS_GKE_2.png)

4. You can specify cluster option in form or simply click on “My first cluster” and button Create

    ![](../../_images/PMM_DBaaS_GKE_3.png)

    ![](../../_images/PMM_DBaaS_GKE_4.png)

5. Wait until cluster created

    ![](../../_images/PMM_DBaaS_GKE_5.png)

6. Click on button Connect in a the cluster’s row

    ![](../../_images/PMM_DBaaS_GKE_6.png)

7. Click button Run in Cloud shell

    ![](../../_images/PMM_DBaaS_GKE_7.png)

8. Click Authorize

    ![](../../_images/PMM_DBaaS_GKE_8.png)

    ![](../../_images/PMM_DBaaS_GKE_9.png)

    ![](../../_images/PMM_DBaaS_GKE_10.png)

9. Set up PXC and PSMDB operators:

    ```
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/bundle.yaml  | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/secrets.yaml | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/bundle.yaml  | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/secrets.yaml | kubectl apply -f -
    ```

    ![](../../_images/PMM_DBaaS_GKE_11.png)

10. Check if it was set up successfully

    ```
    kubectl api-resources --api-group='psmdb.percona.com'
    kubectl api-resources --api-group='pxc.percona.com'
    ```

    ![](../../_images/PMM_DBaaS_GKE_12.png)

11. Check versions

    ```
    kubectl api-versions | grep percona.com
    ```

    ![](../../_images/PMM_DBaaS_GKE_13.png)

12. Create Service Account, copy and store kubeconfig - output of the following command

    ```
    cat <<EOF | kubectl apply -f -
    ---
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: percona-dbaas-cluster-operator
    ---
    kind: RoleBinding
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: service-account-percona-server-dbaas-xtradb-operator
    subjects:
    - kind: ServiceAccount
      name: percona-dbaas-cluster-operator
    roleRef:
      kind: Role
      name: percona-xtradb-cluster-operator
      apiGroup: rbac.authorization.k8s.io
    ---
    kind: RoleBinding
    apiVersion: rbac.authorization.k8s.io/v1beta1
    metadata:
      name: service-account-percona-server-dbaas-psmdb-operator
    subjects:
    - kind: ServiceAccount
      name: percona-dbaas-cluster-operator
    roleRef:
      kind: Role
      name: percona-server-mongodb-operator
      apiGroup: rbac.authorization.k8s.io
    EOF

    name=`kubectl get serviceAccounts percona-dbaas-cluster-operator -o json | jq  -r .secrets[].name`
    certificate=`kubectl get secret $name -o json | jq -r  '.data."ca.crt"'`
    token=`kubectl get secret $name -o json | jq -r  '.data.token' | base64 -d`
    server=`kubectl cluster-info | grep 'Kubernetes master' | cut -d ' ' -f 6`
    ```

    ![](../../_images/PMM_DBaaS_GKE_14.png)


    ```
    echo "
    apiVersion: v1
    kind: Config
    users:
    - name: percona-dbaas-cluster-operator
      user:
        token: $token
    clusters:
    - cluster:
        certificate-authority-data: $certificate
        server: $server
      name: self-hosted-cluster
    contexts:
    - context:
        cluster: self-hosted-cluster
        user: percona-dbaas-cluster-operator
      name: svcs-acct-context
    current-context: svcs-acct-context
    "
    ```

    ![](../../_images/PMM_DBaaS_GKE_15.png)

13. Start PMM Server on you local machine or other VM instance:

    ```
    docker run --detach --name pmm-server --publish 80:80 --publish 443:443 \
    --env PERCONA_TEST_DBAAS=1 perconalab/pmm-server-fb:PR-1240-07bef94;
    ```

14.  Login into PMM and navigate to DBaaS

     ![](../../_images/PMM_DBaaS_GKE_16.png)

15. Register your GKE using kubeconfig from step 12.

    !!! alert alert-warning "Important"
        Please make sure there are no stray new lines in the kubeconfig, especially in long lines like certificate or token.

    ![](../../_images/PMM_DBaaS_GKE_17.png)

    ![](../../_images/PMM_DBaaS_GKE_18.png)

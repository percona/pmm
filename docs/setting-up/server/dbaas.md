# Setting up a development environment for DBaaS

!!! alert alert-warning "Caution"
    DBaaS functionality is Alpha. The information on this page is subject to change and may be inaccurate.

---

[TOC]

---

## Software prerequisites

### Docker

**Red Hat, CentOS**

```sh
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
yum -y install docker-ce
usermod -a -G docker centos
systemctl enable docker
systemctl start docker
```

**Debian, Ubuntu**

```sh
apt-add-repository https://download.docker.com/linux/centos/docker-ce.repo
systemctl enable docker
systemctl start docker
```

### minikube

Please install minukube v1.16.0

**Red Hat, CentOS**

```sh
yum -y install curl
curl -Lo /usr/local/sbin/minikube https://github.com/kubernetes/minikube/releases/download/v1.16.0/minikube-linux-amd64
chmod +x /usr/local/sbin/minikube
ln -s /usr/local/sbin/minikube /usr/sbin/minikube
alias kubectl='minikube kubectl --'
```

## Start PMM server with DBaaS activated

!!! alert alert-info "Notes"
    - To start a fully-working 3 node XtraDB cluster, consisting of sets of 3x ProxySQL, 3x PXC and 6x PMM Client containers, you will need at least 9 vCPU available for minikube. (1x vCPU for ProxySQL and PXC and 0.5vCPU for each pmm-client containers).
    - DBaaS does not depend on PMM Client.
    - Setting the environment variable `PERCONA_TEST_DBAAS=1` enables DBaaS functionality.
    - Add the option `--network minikube` if you run PMM Server and minikube in the same Docker instance. (This will share a single network and the kubeconfig will work.)
    - Add the options `--env PMM_DEBUG=1` and/or `--env PMM_TRACE=1` if you need extended debug details

1. Start PMM server:

    ```sh
    docker run --detach --publish 80:80 --publish 443:443 --name pmm-server --env PERCONA_TEST_DBAAS=1 percona/pmm-server:2
    ```

2. Change the default administrator credentials from CLI:

    (This step is optional, because the same can be done from the web interface of PMM on first login.)

    ```sh
    docker exec -t pmm-server bash -c 'ln -s /srv/grafana /usr/share/grafana/data; chown -R grafana:grafana /usr/share/grafana/data; grafana-cli --homepath /usr/share/grafana admin reset-admin-password <RANDOM_PASS_GOES_IN_HERE>'
    ```

## Install Percona operators in minikube

1. Configure and start minikube:

    ```sh
    minikube config set cpus 16
    minikube config set memory 32768
    minikube config set kubernetes-version 1.16.15
    minikube start
    ```

2. Deploy the Percona operators configuration for PXC and PSMDB in minikube:

    ```sh
    # Prepare a set of base64 encoded values and non encoded for user and pass with administrator privileges to pmm-server (DBaaS)
    PMM_USER='admin';
    PMM_PASS='<RANDOM_PASS_GOES_IN_HERE>';

    PMM_USER_B64="$(echo -n "${PMM_USER}" | base64)";
    PMM_PASS_B64="$(echo -n "${PMM_PASS}" | base64)";

    # Install the PXC operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/bundle.yaml \
    | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/secrets.yaml \
    | sed "s/pmmserver:.*=/pmmserver: ${PMM_PASS_B64}/g" \
    | kubectl apply -f -

    # Install the PSMDB operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.6.0/deploy/bundle.yaml \
    | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.6.0/deploy/secrets.yaml \
    | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER: ${PMM_USER}/g;s/PMM_SERVER_PASSWORD:.*$/PMM_SERVER_PASSWORD: ${PMM_PASS}/g;" \
    | kubectl apply -f -
    ```

3. Check the operators are deployed:

    ```sh
    minikube kubectl -- get nodes
    minikube kubectl -- get pods
    minikube kubectl -- wait --for=condition=Available deployment percona-xtradb-cluster-operator
    minikube kubectl -- wait --for=condition=Available deployment percona-server-mongodb-operator
    ```

4. Get your kubeconfig details from minikube (to register your Kubernetes cluster with PMM Server):

    ```sh
    minikube kubectl -- config view --flatten --minify
    ```
!!! alert alert-info "Note"
    You will need to copy this output to your clipboard and continue with [add a Kubernetes cluster to PMM](../../using/platform/dbaas.md#add-a-kubernetes-cluster).

## Installing Percona operators in AWS EKS (Kubernetes)

1. Create your cluster via `eksctl` or the Amazon AWS interface. For example:

    ```sh
    eksctl create cluster --write-kubeconfig --name=your-cluster-name --zones=us-west-2a,us-west-2b --kubeconfig <PATH_TO_KUBECONFIG>
    ```

2. When your EKS cluster is running, install the PXC and PSMDB operators:

    ```sh
    # Prepare a set of base64 encoded values and non encoded for user and pass with administrator privileges to pmm-server (DBaaS)
    PMM_USER='admin';
    PMM_PASS='<RANDOM_PASS_GOES_IN_HERE>';

    PMM_USER_B64="$(echo -n "${PMM_USER}" | base64)";
    PMM_PASS_B64="$(echo -n "${PMM_PASS}" | base64)";

    # Install the PXC operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/bundle.yaml \
    | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/secrets.yaml \
    | sed "s/pmmserver:.*=/pmmserver: ${PMM_PASS_B64}/g" \
    | kubectl apply -f -

    # Install the PSMDB operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.6.0/deploy/bundle.yaml \
    | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.6.0/deploy/secrets.yaml \
    | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER: ${PMM_USER}/g;s/PMM_SERVER_PASSWORD:.*$/PMM_SERVER_PASSWORD: ${PMM_PASS}/g;" \
    | kubectl apply -f -
    ```

    ```
    # Validate that the operators are running
    kubectl get pods
    ```

3. Modify your kubeconfig file, if it's not utilizing the `aws-iam-authenticator` or `client-certificate` method for authentication with Kubernetes. Here are two examples that you can use as templates to modify a copy of your existing kubeconfig:

    - For the `aws-iam-authenticator` method:

        ```yml
        ---
        apiVersion: v1
        clusters:
        - cluster:
            certificate-authority-data: << CERT_AUTH_DATA >>
            server: << K8S_CLUSTER_URL >>
          name: << K8S_CLUSTER_NAME >>
        contexts:
        - context:
            cluster: << K8S_CLUSTER_NAME >>
            user: << K8S_CLUSTER_USER >>
          name: << K8S_CLUSTER_NAME >>
        current-context: << K8S_CLUSTER_NAME >>
        kind: Config
        preferences: {}
        users:
        - name: << K8S_CLUSTER_USER >>
          user:
            exec:
              apiVersion: client.authentication.k8s.io/v1alpha1
              command: aws-iam-authenticator
              args:
                - "token"
                - "-i"
                - "<< K8S_CLUSTER_NAME >>"
                - --region
                - << AWS_REGION >>
              env:
                 - name: AWS_ACCESS_KEY_ID
                   value: "<< AWS_ACCESS_KEY_ID >>"
                 - name: AWS_SECRET_ACCESS_KEY
                   value: "<< AWS_SECRET_ACCESS_KEY >>"
        ```

     - For the `client-certificate` method:

        ```yml
        ---
        apiVersion: v1
        clusters:
        - cluster:
            certificate-authority-data: << CERT_AUTH_DATA >>
            server: << K8S_CLUSTER_URL >>
          name: << K8S_CLUSTER_NAME >>
        contexts:
        - context:
            cluster: << K8S_CLUSTER_NAME >>
            user: << K8S_CLUSTER_USER >>
          name: << K8S_CLUSTER_NAME >>
        current-context: << K8S_CLUSTER_NAME >>
        kind: Config
        preferences: {}
        users:
        - name: << K8S_CLUSTER_NAME >>
          user:
            client-certificate-data: << CLIENT_CERT_DATA >>
            client-key-data: << CLIENT_KEY_DATA >>
        ```

4. Follow the instructions for [Add a Kubernetes cluster](../../using/platform/dbaas.md#add-a-kubernetes-cluster).

!!! alert alert-info "Note"
    If possible, the connection details will show the cluster's external IP (not possible with minikube).



{% include 'setting-up/server/dbaas-gke.md' %}

## Deleting clusters

You should delete all installation operators as the operators own resources.

If you only run `eksctl delete cluster` without cleaning up the cluster first, there will be a lot of orphaned resources as Cloud Formations, Load Balancers, EC2 instances, Network interfaces, etc.

In the `pmm-managed` repository, in the deploy directory there are 2 example bash scripts to install and delete the operators from the EKS cluster.

The install script:

```sh
#!/bin/bash

TOP_DIR=$(git rev-parse --show-toplevel)
PMM_USER="$(echo -n 'admin' | base64)";
PMM_PASS="$(echo -n 'admin_password' | base64)";
KUBECTL_CMD="kubectl --kubeconfig ${HOME}/.kube/config_eks"

# Install the PXC operator
cat ${TOP_DIR}/deploy/pxc_operator.yaml | ${KUBECTL_CMD} apply -f -
cat ${TOP_DIR}/deploy/secrets.yaml | sed "s/pmmserver:.*=/pmmserver: ${PMM_PASS}/g" | ${KUBECTL_CMD} apply -f -

# Install the PSMDB operator
cat ${TOP_DIR}/deploy/psmdb_operator.yaml | ${KUBECTL_CMD} apply -f -
cat ${TOP_DIR}/deploy/secrets.yaml | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER: ${PMM_USER}/g;s/PMM_SERVER_PASSWORD:.*=$/PMM_SERVER_PASSWORD: ${PMM_PASS}/g;" | ${KUBECTL_CMD} apply -f -
```

The delete script:

```sh
#!/bin/bash

TOP_DIR=$(git rev-parse --show-toplevel)
PMM_USER="$(echo -n 'admin' | base64)";
PMM_PASS="$(echo -n 'admin_password' | base64)";
KUBECTL_CMD="kubectl --kubeconfig ${HOME}/.kube/config_eks"

# Delete the PXC operator
cat ${TOP_DIR}/deploy/pxc_operator.yaml | ${KUBECTL_CMD} delete -f -
cat ${TOP_DIR}/deploy/secrets.yaml | sed "s/pmmserver:.*=/pmmserver: ${PMM_PASS}/g" | ${KUBECTL_CMD} delete -f -

# Delete the PSMDB operator
cat ${TOP_DIR}/deploy/psmdb_operator.yaml | ${KUBECTL_CMD} delete -f -
cat ${TOP_DIR}/deploy/secrets.yaml | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER: ${PMM_USER}/g;s/PMM_SERVER_PASSWORD:.*=$/PMM_SERVER_PASSWORD: ${PMM_PASS}/g;" | ${KUBECTL_CMD} delete -f -
```

(Both scripts are similar except the install script command is `apply` while in the delete script it is `delete`.)

After deleting everything in the EKS cluster, run this command (using your own configuration path) and wait until the output only shows service/kubernetes before deleting the cluster with the `eksclt delete` command.

```sh
kubectl --kubeconfig ~/.kube/config_eks get all
```

Example output:

```
NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   10.100.0.1   <none>        443/TCP   4d5h
```

If you don't need the cluster anymore, you can uninstall everything in it and destroy it:

```sh
# Delete all volumes created by the operators:
kubectl [--kubeconfig <config file>] delete pvc --all
# Delete the cluster
eksctl delete cluster --name=your-cluster-name
```

## Run PMM Server as a Docker container for DBaaS

1. Start PMM server from a feature branch:

    ```sh
    docker run --detach --name pmm-server --publish 80:80 --publish 443:443 --env PERCONA_TEST_DBAAS=1  percona/pmm-server:2;
    ```

    !!! alert alert-warning "Important"
        - Use `--network minikube` if running PMM Server and minikube in the same Docker instance. This way they will share single network and the kubeconfig will work.
        - Use Docker variables `--env PMM_DEBUG=1 --env PMM_TRACE=1` to see extended debug details.

2. Change the default administrator credentials:

    !!! alert alert-info "Note"
        This step is optional, because the same can be done from the web interface of PMM on the first login.

    ```sh
    docker exec -t pmm-server bash -c 'ln -s /srv/grafana /usr/share/grafana/data; chown -R grafana:grafana /usr/share/grafana/data; grafana-cli --homepath /usr/share/grafana admin reset-admin-password <RANDOM_PASS_GOES_IN_HERE>'
    ```

3. Set the public address for PMM Server in PMM settings UI

4. Follow the steps for [Add a Kubernetes cluster](../../using/platform/dbaas.md#add-a-kubernetes-cluster).

5. Follow the steps for [Add a DB Cluster](../../using/platform/dbaas.md#add-a-db-cluster).

6. Get the IP address to connect your app/service:

    ```sh
    minikube kubectl get services
    ```

## Exposing PSMDB and XtraDB clusters for access by external clients

To make services visible externally, you create a LoadBalancer service or manually run commands to expose ports:

```sh
kubectl expose deployment hello-world --type=NodePort.
```

!!! seealso "See also"
    - [DBaaS Dashboard](../../using/platform/dbaas.md)
    - [Install minikube](https://minikube.sigs.k8s.io/docs/start/)
    - [Setting up a Standalone MYSQL Instance on Kubernetes & exposing it using Nginx Ingress Controller](https://medium.com/@chrisedrego/setting-up-a-standalone-mysql-instance-on-kubernetes-exposing-it-using-nginx-ingress-controller-262fc7af593a)
    - [Use a Service to Access an Application in a Cluster.](https://kubernetes.io/docs/tasks/access-application-cluster/service-access-application-cluster/)
    - [Exposing applications using services.](https://cloud.google.com/kubernetes-engine/docs/how-to/exposing-apps)

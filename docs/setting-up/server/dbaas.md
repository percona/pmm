# Setting up a development environment for DBaaS

!!! note "Caution"
    DBaaS functionality is Alpha. The information on this page is subject to change and may be inaccurate.

## Software prerequisites

### Docker

To install Docker on CentOS:

```sh
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo;
yum -y install docker-ce;
usermod -a -G docker centos;
systemctl enable docker;
systemctl start docker;
```

### minikube

To install minikube on CentOS:

```sh
yum -y install curl;
curl -Lo /usr/local/sbin/minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64  && chmod +x /usr/local/sbin/minikube
ln -s /usr/local/sbin/minikube /usr/sbin/minikube
alias kubectl='minikube kubectl --'
```

## Start PMM server with DBaaS activated

!!! note "Notes"
    - To start a fully-working 3 node XtraDB cluster, consisting of sets of 3x ProxySQL, 3x PXC and 6x PMM Client containers, you will need at least 9 vCPU available for minikube. (1x vCPU for ProxySQL and PXC and 0.5vCPU for each pmm-client containers).
    - DBaaS does not depend on PMM Client.
    - Setting the environment variable `PERCONA_TEST_DBAAS=1` enables DBaaS functionality.
    - Add the option `--network minikube` if you run PMM Server and minikube in the same Docker instance. (This will share a single network and the kubeconfig will work.)
    - Add the options `--env PMM_DEBUG=1` and/or `--env PMM_TRACE=1` if you need extended debug details

1. Start PMM server from a feature branch:

    ```sh
    docker run --detach --publish 80:80 --name pmm-server --env PERCONA_TEST_DBAAS=1 perconalab/pmm-server-fb:<feature branch ID>
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
    # Base64 encoded USER and PASS for pmm-server
    PMM_USER="\$(echo -n 'admin' | base64)";
    PMM_PASS="\$(echo -n '<RANDOM_PASS_GOES_IN_HERE>' | base64)";

    # Deploy PXC operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/bundle.yaml | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/secrets.yaml | sed "s/pmmserver:.*=/pmmserver: \${PMM_PASS}/g" | kubectl -- apply -f -

    # Deploy PSMDB operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/bundle.yaml | kubectl -- apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/secrets.yaml | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER
    ```

3. Check the operators are deployed:

    ```sh
    minikube kubectl -- get nodes
    minikube kubectl -- get pods
    minikube kubectl -- wait --for=condition=Available deployment percona-xtradb-cluster-operator
    minikube kubectl -- wait --for=condition=Available deployment percona-server-mongodb-operator
    ```

4. Get your kubeconfig details from minikube (to register your k8s cluster with PMM Server):

    ```sh
    minikube kubectl -- config view --flatten --minify
    ```

## Installing Percona operators in AWS EKS (kubernetes)

1. Create your cluster via `eksctl` or the Amazon AWS interface. Example command:

    ```sh
    eksctl create cluster --write-kubeconfig —name=your-cluster-name —zones=us-west-2a,us-west-2b --kubeconfig <PATH_TO_KUBECONFIG>
    ```

2. After your EKS cluster is up you need to install the PXC and PSMDB operators in. This is done the following way:

    ```sh
    # Prepare a base64 encoded values for user and pass with administrator privileges to pmm-server (DBaaS)
    PMM_USER="$(echo -n 'admin' | base64)";
    PMM_PASS="$(echo -n '<RANDOM_PASS_GOES_IN_HERE>' | base64)";

    # Install the PXC operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/bundle.yaml  | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-xtradb-cluster-operator/pmm-branch/deploy/secrets.yaml | sed "s/pmmserver:.*=/pmmserver: ${PMM_PASS}/g" | kubectl apply -f -

    # Install the PSMDB operator
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/bundle.yaml  | kubectl apply -f -
    curl -sSf -m 30 https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/pmm-branch/deploy/secrets.yaml | sed "s/PMM_SERVER_USER:.*$/PMM_SERVER_USER: ${PMM_USER}/g;s/PMM_SERVER_PASSWORD:.*=$/PMM_SERVER_PASSWORD: ${PMM_PASS}/g;" | kubectl apply -f -

    # Validate that the operators are running
    kubectl get pods
    ```

3. Then you need to modify your kubeconfig file, if it's not utilizing the `aws-iam-authenticator` or `client-certificate` method for authentication against k8s. Here are two examples that you can use as template to modify a copy of your existing kubeconfig:

    - For `aws-iam-authenticator` method:

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

     - For `client-certificate` method:

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

!!! seealso "See also"
    - [DBaaS Dashboard](../../using/platform/dbaas.md)
    - [Install minikube](https://minikube.sigs.k8s.io/docs/start/)

# Run PMM Client as a Pod in a Kubernetes Deployment

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) can be deployed as a pod in Kubernetes, provides a convenient way to run PMM Client as a pre-configured container without installing software directly on your host system.

Using the Kubernetes Pod approach offers several advantages:

- no need to install PMM Client directly on your host system
- consistent environment across different operating systems
- simplified setup and configuration process
- automatic architecture detection (x86_64/ARM64)
- [centralized configuration management](../install-pmm-server/deployment-options/docker/env_var.md#configure-vmagent-variables) through PMM Server environment variables

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/).

2. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

3. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll need its IP address or hostname to configure the Client.

4. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

5. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

## Installation and setup

### Deploy PMM Client

=== "Deploy PMM Client as a Standalone container"
    Follow these steps to deploy PMM Client using `kubectl`:
    {.power-number}

    1. (Optional) Create a namespace named `pmm-client-test` for the deployment and set it as the default namespace:

        ```sh
        kubectl create namespace pmm-client-test
        kubectl config set-context --current --namespace=pmm-client-test
        ```

    2. Create the file `pmm-client-volume.yaml` with the following content to define a Persistent Volume for storing PMM Client data between pod restarts:

        ```yaml
        apiVersion: v1
        kind: PersistentVolume
        metadata:
          name: pmm-client-pv
          labels:
            type: local
        spec:
          storageClassName: manual
          capacity:
            storage: 10Gi
          accessModes:
            - ReadWriteOnce
          hostPath:
            path: "/mnt/data"
        ---
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          name: pmm-client-pvc
        spec:
          storageClassName: manual
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 10Gi
        ```

    3. Create the resources defined in `pmm-client-volume.yaml`

        ```sh
        kubectl apply -f pmm-client-volume.yaml
        ```

    4. Create a Secret to store the credentials for PMM Server authentication. Update `PMM_AGENT_SERVER_PASSWORD` value if you changed the default `admin` password during setup:

        ```sh
        kubectl create secret generic pmm-secret \
        --from-literal=PMM_AGENT_SERVER_USERNAME=admin \
        --from-literal=PMM_AGENT_SERVER_PASSWORD=admin
        ```

    5. Create the file `pmm-client-pod.yaml` with the following content to define a Pod running PMM Client. Replace `X.X.X.X` with the IP address of your PMM Server:

        ```yaml
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: pmm-client
        spec:
          selector:
            matchLabels:
              app: pmm-client
          strategy:
            type: Recreate
          template:
            metadata:
              labels:
                app: pmm-client
            spec:
              containers:
                - name: pmm-client
                  image: percona/pmm-client:3
                  volumeMounts:
                    - name: pmm-client-pvc
                      mountPath: /usr/local/percona/pmm/tmp
                  env:
                    - name: PMM_AGENT_SERVER_ADDRESS
                      value: X.X.X.X:443
                    - name: PMM_AGENT_SERVER_USERNAME
                      valueFrom:
                        secretKeyRef:
                          name: pmm-secret
                          key: PMM_AGENT_SERVER_USERNAME
                    - name: PMM_AGENT_SERVER_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: pmm-secret
                          key: PMM_AGENT_SERVER_PASSWORD
                    - name: PMM_AGENT_SERVER_INSECURE_TLS
                      value: "1"
                    - name: PMM_AGENT_CONFIG_FILE
                      value: config/pmm-agent.yaml
                    - name: PMM_AGENT_SETUP
                      value: "1"
                    - name: PMM_AGENT_SETUP_FORCE
                      value: "1"
              volumes:
                - name: pmm-client-storage
                  persistentVolumeClaim:
                    claimName: pmm-client-pvc
        ```

    6. Deploy PMM Client pod and configure the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server:

        ```sh
        kubectl apply -f pmm-client-pod.yaml
        ```

        !!! hint alert-success "Important"
            - You can set the container environment variable `PMM_AGENT_PRERUN_SCRIPT` to a shell script so that it will automatically add service(s) to PMM for monitoring.
            - If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable.

=== "Deploy PMM Client as a Sidecar container"
    Follow these steps to deploy PMM Client as a Sidecar container to a MySQL container using `kubectl`:
    {.power-number}

    1. (Optional) Create a namespace named `pmm-client-test` for the deployment and set it as the default namespace:

        ```sh
        kubectl create namespace pmm-client-test
        kubectl config set-context --current --namespace=pmm-client-test
        ```

    2. Create the file `mysql-pmm-client-volume.yaml` with the following content to define a Persistent Volume for storing PMM Client and MySQL data between pod restarts:

        ```yaml
        apiVersion: v1
        kind: PersistentVolume
        metadata:
          name: pmm-client-pv
          labels:
            type: local
        spec:
          storageClassName: manual
          capacity:
            storage: 10Gi
          accessModes:
            - ReadWriteOnce
          hostPath:
            path: "/mnt/data/pmm-client"
        ---
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          name: pmm-client-pvc
        spec:
          storageClassName: manual
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 10Gi
        ---
        apiVersion: v1
        kind: PersistentVolume
        metadata:
          name: mysql-pv-volume
          labels:
            type: local
        spec:
          storageClassName: manual
          capacity:
            storage: 20Gi
          accessModes:
            - ReadWriteOnce
          hostPath:
            path: "/mnt/data/mysql"
        ---
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          name: mysql-pv-claim
        spec:
          storageClassName: manual
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 20Gi
        ```

    3. Create the resources defined in `mysql-pmm-client-volume.yaml`

        ```sh
        kubectl apply -f mysql-pmm-client-volume.yaml
        ```

    4. Create a Secret to store the credentials for PMM Server authentication. Update `PMM_AGENT_SERVER_PASSWORD` value if you changed the default `admin` password during setup:

        ```sh
        kubectl create secret generic pmm-secret \
         --from-literal=PMM_AGENT_SERVER_USERNAME=admin \
         --from-literal=PMM_AGENT_SERVER_PASSWORD=admin
        ```

    5. Create a Secret to store the MySQL root password:

        ```sh
        kubectl create secret generic mysql-secret \
         --from-literal=MYSQL_ROOT_PASSWORD=very_secure_password
        ```
    6. Create the file `mysql-pmm-client-pod.yaml` with the following content to define a Pod running MySQL 9.0 container with a PMM Client container running as Sidecar. Replace `X.X.X.X` with the IP address of your PMM Server:

        ```yaml
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: mysql
        spec:
          selector:
            matchLabels:
              app: mysql
          strategy:
            type: Recreate
          template:
            metadata:
              labels:
                app: mysql
            spec:
              containers:
                - name: mysql
                  image: mysql:9
                  resources: {}
                  env:
                    # Use secret in real usage
                    - name: MYSQL_ROOT_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: mysql-secret
                          key: MYSQL_ROOT_PASSWORD
                  ports:
                    - containerPort: 3306
                      name: mysql
                  volumeMounts:
                    - name: mysql-persistent-storage
                      mountPath: /var/lib/mysql
                - name: pmm-client
                  image: percona/pmm-client:3
                  env:
                    - name: PMM_AGENT_SERVER_ADDRESS
                      value: X.X.X.X:443
                    - name: PMM_AGENT_SERVER_USERNAME
                      valueFrom:
                        secretKeyRef:
                          name: pmm-secret
                          key: PMM_AGENT_SERVER_USERNAME
                    - name: PMM_AGENT_SERVER_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: pmm-secret
                          key: PMM_AGENT_SERVER_PASSWORD
                    - name: MYSQL_ROOT_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: mysql-secret
                          key: MYSQL_ROOT_PASSWORD
                    - name: PMM_AGENT_SERVER_INSECURE_TLS
                      value: "1"
                    - name: PMM_AGENT_CONFIG_FILE
                      value: config/pmm-agent.yaml
                    - name: PMM_AGENT_SETUP
                      value: "1"
                    - name: PMM_AGENT_SETUP_FORCE
                      value: "1"
                    - name: PMM_AGENT_SIDECAR
                      value: "1"
                    - name: PMM_AGENT_PRERUN_SCRIPT
                      value: "pmm-admin status --wait=10s; pmm-admin add mysql --username=root --password=${MYSQL_ROOT_PASSWORD} --query-source=perfschema"
              volumes:
                - name: mysql-persistent-storage
                  persistentVolumeClaim:
                    claimName: mysql-pv-claim
                - name: pmm-client-storage
                  persistentVolumeClaim:
                    claimName: pmm-client-pvc
        ```


    6. Deploy MySQL and PMM Client pod:

        ```sh
        kubectl apply -f mysql-pmm-client-pod.yaml
        ```

        !!! hint alert-success "Important"
            - If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable.


## View your monitored node

To confirm your node is being monitored:
{.power-number}

1. Go to the main menu and select **Operating System (OS) > Overview**.

2. In the **Node Names** drop-down menu, select the node you recently registered.

3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.

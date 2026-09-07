# Run PMM Client as a Kubernetes pod

Deploy the [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) as a Kubernetes pod to monitor your databases without installing software on your host system. 

This deployment approach provides:

- automatic architecture detection (x86_64/ARM64)
- consistent environment across different operating systems
- simplified setup and configuration
- [centralized configuration management](../install-pmm-server/deployment-options/docker/env_var.md#configure-vmagent-variables) via PMM Server

## Prerequisites

Before deploying PMM Client:
{.power-number}

1. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/).

2. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

3. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll need its IP address or hostname to configure the Client.

4. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

5. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

## Installation and setup

### Deploy PMM Client

Choose your deployment approach:

- **Standalone**: Deploy PMM Client as a dedicated pod to monitor external databases
- **Sidecar**: Deploy PMM Client alongside a database in the same pod 

=== "Deploy PMM Client as a Standalone container"
    Deploy PMM Client as a StatefulSet, so that the pod keeps its name when it restarts. PMM
    identifies a Node by name, and the Agent identity that PMM Server issues is stored in a file on
    the pod's volume, so both the name and the volume have to be stable for the Services you add to
    survive a restart or an image upgrade.

    !!! caution alert alert-warning "Requires PMM Client 3.10.0 or later"
        Earlier versions register the Node again on every container start, and PMM Server answers a
        re-registration by removing the Node together with every Service configured on it. On those
        versions the Services you add to a restarted pod are lost regardless of the storage you give it.

    Follow these steps to deploy PMM Client using `kubectl`:
    {.power-number}

    1. (Optional) Create a namespace for the deployment and set it as the default namespace:

        ```sh
        kubectl create namespace pmm-client-test
        kubectl config set-context --current --namespace=pmm-client-test
        ```

    2. Create a Secret to store PMM Server credentials. Replace `admin` password if you changed it during PMM Server setup:

        ```sh
        kubectl create secret generic pmm-secret \
        --from-literal=PMM_AGENT_SERVER_USERNAME=admin \
        --from-literal=PMM_AGENT_SERVER_PASSWORD=admin
        ```

    3. Create `pmm-client.yaml` to define the StatefulSet and its volume. Replace `X.X.X.X` with the IP address of your PMM Server:

        ```yaml
        apiVersion: apps/v1
        kind: StatefulSet
        metadata:
          name: pmm-client
        spec:
          serviceName: pmm-client
          replicas: 1
          selector:
            matchLabels:
              app: pmm-client
          template:
            metadata:
              labels:
                app: pmm-client
            spec:
              securityContext:
                # The PMM Client image runs as this user, which has to own the volume to write to it.
                fsGroup: 1002
              containers:
                - name: pmm-client
                  image: percona/pmm-client:3
                  volumeMounts:
                    # Holds the Agent identity, so the pod keeps its Node and the Services on it.
                    - name: pmm-agent
                      mountPath: /usr/local/percona/pmm/config
                      subPath: config
                    # Holds the metrics buffered on disk while PMM Server is unreachable.
                    - name: pmm-agent
                      mountPath: /usr/local/percona/pmm/tmp
                      subPath: tmp
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
                    # An absolute path, so that it resolves to the mounted volume whatever the
                    # container's working directory is.
                    - name: PMM_AGENT_CONFIG_FILE
                      value: /usr/local/percona/pmm/config/pmm-agent.yaml
                    - name: PMM_AGENT_SETUP
                      value: "1"
                    - name: PMM_AGENT_SETUP_NODE_TYPE
                      value: container
                    # The pod name of a StatefulSet replica does not change when the pod restarts,
                    # which is what lets PMM Server recognise it as the same Node.
                    - name: PMM_AGENT_SETUP_NODE_NAME
                      valueFrom:
                        fieldRef:
                          fieldPath: metadata.name
          volumeClaimTemplates:
            - metadata:
                name: pmm-agent
              spec:
                accessModes:
                  - ReadWriteOnce
                resources:
                  requests:
                    storage: 2Gi
        ```

        !!! note alert alert-primary "Storage"
            The volume is claimed from your cluster's default StorageClass. Add `storageClassName` to
            the claim template to choose a different one. Keep the size above 1Gi: that is the
            on-disk queue the Client fills with metrics it cannot deliver while PMM Server is
            unreachable.

        !!! warning alert alert-warning "Security and configuration"
            - The `PMM_AGENT_SERVER_INSECURE_TLS=1` setting disables TLS certificate verification. For production environments, configure proper TLS certificates and remove this setting.
            - If disk metrics appear missing or incorrect, your container may not expose `/proc/mounts` at the default path. Add `PMM_AGENT_SETUP_PROC_MOUNTS_PATH` to the `env` section to point PMM Client to the correct location:

            ```yaml
            - name: PMM_AGENT_SETUP_PROC_MOUNTS_PATH
              value: /path/to/proc/mounts
            ```

    4. Deploy PMM Client and configure the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server:

        ```sh
        kubectl apply -f pmm-client.yaml
        ```

    5. Check that the Node registered, and that a restart keeps it:

        ```sh
        kubectl logs pmm-client-0
        kubectl delete pod pmm-client-0
        kubectl logs pmm-client-0
        ```

        On the first start the Client registers the Node. After the restart it reports that the Node
        is registered already and keeps its identity, so any Services you added remain.

    !!! hint alert-success "Adding services automatically"
        You can set the container environment variable `PMM_AGENT_PRERUN_SCRIPT` to a shell script to automatically add services to PMM for monitoring. Because this deployment keeps the Services you add, a prerun script is only needed if you would rather declare them in the manifest than add them once.

    !!! caution alert alert-warning "Recovering a Node after losing the volume"
        If the volume is deleted while the Node still exists in PMM, the Client cannot register that
        Node name again and the pod fails to start. Add `PMM_AGENT_SETUP_FORCE=1` to the `env`
        section for one start to take the name over. PMM Server then removes the old Node together
        with every Service on it, so remove the variable again afterwards.


=== "Deploy PMM Client as a Sidecar container"
    Follow these steps to deploy PMM Client as a Sidecar container to a MySQL container using `kubectl`:
    {.power-number}

    1. (Optional) Create a namespace named `pmm-client-test` for the deployment and set it as the default namespace:

        ```sh
        kubectl create namespace pmm-client-test
        kubectl config set-context --current --namespace=pmm-client-test
        ```

    2. Create `mysql-pmm-client-volume.yaml` to define persistent storage for storing PMM Client and MySQL data between pod restarts:

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
    
    6. Create `mysql-pmm-client-pod.yaml` to define a Pod running MySQL 9.0 container with a PMM Client container running as Sidecar. Replace `X.X.X.X` with the IP address of your PMM Server:

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

        !!! warning alert alert-warning "Security note"
            The `PMM_AGENT_SERVER_INSECURE_TLS=1` setting disables TLS certificate verification. For production environments, configure proper TLS certificates and remove this setting.

    7. Deploy MySQL and PMM Client pod:

          ```sh
          kubectl apply -f mysql-pmm-client-pod.yaml
          ```

## View your monitored node

To confirm your node is being monitored:
{.power-number}

1. Go to the main menu and select **Operating system > Overview**.

2. In the **Node Names** drop-down menu, select the node you recently registered.

3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.

## Troubleshooting

### Failed to register pmm-agent on PMM Server: connection refused

If you get `Failed to register pmm-agent on PMM Server: connection refused`, this typically means that the IP address is incorrect or the PMM Server is unreachable. Verify:

- The `PMM_AGENT_SERVER_ADDRESS` value is correct
- PMM Server is running and accessible
- Firewall rules allow traffic on port `443`

### Pod stuck in Pending state
Check that the volume was bound. The standalone StatefulSet claims it through its volume claim
template, which names the claim after the template and the pod:

```sh
kubectl get pvc
kubectl describe pvc pmm-agent-pmm-client-0
```

A claim left `Pending` usually means the cluster has no default StorageClass. List the available
classes with `kubectl get storageclass` and set `storageClassName` in the claim template.

### View PMM Client logs

```sh
# Standalone deployment
kubectl logs -l app=pmm-client

# Sidecar deployment
kubectl logs -l app=mysql -c pmm-client
```
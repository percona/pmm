# Install PMM on a Kubernetes single instance

Install PMM Server as a single pod on Kubernetes using the `percona/pmm` Helm chart. Kubernetes restarts it automatically if it fails, and persistent volumes keep your data across restarts. 

If 2–5 minutes of monitoring downtime during pod rescheduling is not acceptable, consider [PMM HA Cluster](install-HA-clustered.md) instead.

## Prerequisites

Check that your environment meets the software and cluster resource requirements before installing:

=== "Required software"

    Required tools and components:

    - **Kubernetes**: 1.21 or higher
    - **Helm**: 3.2.0 or higher
    - **kubectl**: Configured to access your cluster
    - **Persistent Volume Provisioner**: Available in your cluster (e.g., AWS EBS, GCE PD, Azure Disk)

=== "Cluster requirements"

    Minimum cluster resources:

    - **CPU**: 2 cores available
    - **Memory**: 4 GB RAM available
    - **Storage**: 20+ GB persistent volume

=== "Recommended for production"

    Recommended resources for production use:

    - **CPU**: 4+ cores
    - **Memory**: 8+ GB RAM
    - **Storage**: 100+ GB with fast SSD-backed persistent volumes
    - **Multiple nodes**: At least 2 worker nodes for automatic rescheduling


### Supported platforms

The `percona/pmm` chart runs on any Kubernetes 1.21 or later and is tested on Red Hat OpenShift 4.18 to 4.22, including ROSA with hosted control planes. Other platforms such as GKE, AKS, and on-premises Kubernetes may work but aren't tested.

### Verify prerequisites

Check your cluster meets the requirements:
```sh
# Check Kubernetes version
kubectl version

# Check available resources
kubectl top nodes

# Verify storage classes
kubectl get storageclass

# Check Helm version
helm version --short
```

## Install PMM Server

Choose the installation method that fits your needs and install PMM Server on Kubernetes.

=== "Quick start"

    Get PMM Server running with default settings:
    {.power-number}

    1. Add the Percona Helm repository:
      ```sh
        helm repo add percona https://percona.github.io/percona-helm-charts/
        helm repo update
      ```

    2. Create a namespace for PMM:
      ```sh
        kubectl create namespace monitoring
      ```

    3. Install PMM Server:
      ```sh
        helm install pmm percona/pmm \
          --namespace monitoring \
          --set service.type=LoadBalancer
      ```

    4. Get the external IP address:
      ```sh
        kubectl get svc -n monitoring monitoring-service
      ```

        The external IP may take a minute to provision. Look for the `EXTERNAL-IP` column.

    5. Get the admin password that the chart generated:
      ```sh
        kubectl get secret pmm-secret -n monitoring -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
      ```

    6. Open `https://<EXTERNAL-IP>` in your browser and log in as `admin` with that password.

=== "Recommended"

    Customize your deployment with storage, resource, and security settings:
    {.power-number}

    1. Create a `values.yaml` file with your configuration:
      ```yaml
        service:
          type: LoadBalancer
          annotations:
            # AWS example - use NLB for better performance
            service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
            # GCP example - use internal LB for VPC-only access
            # cloud.google.com/load-balancer-type: "Internal"

        storage:
          storageClassName: "fast-ssd"  # Use your fastest storage class
          size: 100Gi

        pmmResources:
          requests:
            memory: 4Gi
            cpu: 2
          limits:
            memory: 8Gi
            cpu: 4

        # Configure data retention (default: 30 days)
        pmmEnv:
          PMM_DATA_RETENTION: "720h"  # 30 days

        # Configure security
        secret:
          # Change default password (or use existing secret)
          pmm_password: "your-secure-password-here"
      ```

    2. Add the Percona Helm repository:
      ```sh
        helm repo add percona https://percona.github.io/percona-helm-charts/
        helm repo update
      ```

    3. Create a namespace for PMM:
      ```sh
        kubectl create namespace monitoring
      ```

    4. Install PMM Server with your custom values:
      ```sh
        helm install pmm percona/pmm \
          --namespace monitoring \
          --values values.yaml
      ```

    5. Get the external IP address:
      ```sh
        kubectl get svc -n monitoring monitoring-service
      ```

        Look for the `EXTERNAL-IP` column. The IP may take a minute to provision.

    6. Open `https://<EXTERNAL-IP>` in your browser and log in with the password you set in `values.yaml`.

### Verify installation

Regardless of which method you chose, verify that PMM Server is running correctly:
```sh
# Check pod status
kubectl get pods -n monitoring

# Check persistent volume claim
kubectl get pvc -n monitoring

# Check service
kubectl get svc -n monitoring

# View pod logs
kubectl logs -n monitoring -l app.kubernetes.io/name=pmm

# Wait for pod to be ready
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=pmm \
  -n monitoring \
  --timeout=300s
```

Expected output:
```
pod/pmm-0   1/1   Running   0   2m
```

## Configure PMM Server

After installation, set the admin password, expose the service externally, choose a storage class, adjust data retention, and set resource limits to match your environment.

### Change admin password

After first login, immediately change the default password:
{.power-number}

1. Log in to PMM UI.
2. Go to **Account > Change password**.
4. Enter current password and new secure password.

Or set password via Kubernetes secret before installation:
```sh
# Create secret with custom password
kubectl create secret generic pmm-secret \
  --from-literal=PMM_ADMIN_PASSWORD='your-secure-password' \
  -n monitoring

# Reference in values.yaml
secret:
  name: pmm-secret
```

### Configure external access

By default, PMM Server is only accessible inside the cluster. To reach it from a browser or from PMM Clients running outside the cluster, expose it using a Kubernetes Service. Choose the service type that fits your environment:

=== "LoadBalancer"

    **Best for**: Cloud environments (AWS, GCP, Azure)

    Use LoadBalancer for automatic provisioning of external IP addresses:
    {.power-number}

    1. Create or update your `values.yaml` file:
      ```yaml
        service:
          type: LoadBalancer
          annotations:
            # Add cloud-provider-specific annotations
            # AWS example:
            # service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
            # GCP example:
            # cloud.google.com/load-balancer-type: "Internal"
      ```

    2. Apply the configuration:
      ```sh
        helm upgrade pmm percona/pmm \
          --namespace monitoring \
          --values values.yaml
      ```

    3. Get the external IP address:
      ```sh
        kubectl get svc -n monitoring monitoring-service
      ```

        Look for the `EXTERNAL-IP` column and connect via `https://<EXTERNAL-IP>`.

=== "NodePort"

    **Best for**: Bare-metal, on-premise, or testing environments

    Use NodePort to expose PMM on a static port on each cluster node:
    {.power-number}

    1. Create or update your `values.yaml` file:
      ```yaml
        service:
          type: NodePort
          nodePort: 30443  # Optional: specify port (30000-32767 range)
      ```

    2. Apply the configuration:
      ```sh
        helm upgrade pmm percona/pmm \
          --namespace monitoring \
          --values values.yaml
      ```

    3. Get the assigned NodePort:
      ```sh
        kubectl get svc -n monitoring monitoring-service
      ```

        Look for the `PORT(S)` column showing the NodePort number.

    4. Access PMM using any node IP and the NodePort:
      ```
        https://<any-node-ip>:<nodeport>
      ```

=== "Ingress"

    **Best for**: Production environments with existing ingress controller

    **Prerequisites**: An ingress controller (e.g., NGINX, Traefik) and cert-manager installed in your cluster.

    Use Ingress for advanced routing, SSL termination, and custom domain names:
    {.power-number}

    1. Create or update your `values.yaml` file:
      ```yaml
        service:
          type: ClusterIP

        ingress:
          enabled: true
          className: nginx
          annotations:
            cert-manager.io/cluster-issuer: "letsencrypt-prod"
          hosts:
            - host: pmm.example.com
              paths:
                - path: /
                  pathType: Prefix
          tls:
            - secretName: pmm-tls
              hosts:
                - pmm.example.com
      ```

        Replace `pmm.example.com` with your domain.

    2. Apply the configuration:
      ```sh
        helm upgrade pmm percona/pmm \
          --namespace monitoring \
          --values values.yaml
      ```

    3. Access PMM at your configured domain:
      ```
        https://pmm.example.com
      ```



### Configure storage

PMM Server stores metrics, query analytics data, and Grafana settings on a persistent volume. Use your fastest available storage class to avoid I/O bottlenecks as data grows:
{.power-number}

1. Find the storage classes available in your cluster:

    - **AWS**: `gp3` (recommended), `gp2`, `io1`
    - **GCP**: `pd-ssd`, `pd-balanced`
    - **Azure**: `managed-premium`, `managed`
    - **On-premise**: `kubectl get storageclass`

2. Set the storage class in your `values.yaml`:

    ```yaml
    # values.yaml
    storage:
      storageClassName: "fast-ssd"  # Use fastest available
      size: 100Gi

      # Optional: Storage selector
      # selector:
      #   matchLabels:
      #     type: ssd
    ```

3. Install PMM Server with these values. Kubernetes doesn't apply storage changes to the volume of an existing installation, so set them before you install:

    ```sh
    helm install pmm percona/pmm \
      --namespace monitoring \
      --values values.yaml
    ```

### Configure data retention

PMM retains monitoring data for 30 days by default. To change it, set `PMM_DATA_RETENTION` in your `values.yaml` and apply the change:
{.power-number}

1. Set the retention period. Common values:

    - `168h`: 7 days (minimal storage)
    - `720h`: 30 days (recommended)
    - `2160h`: 90 days (compliance/audit)

    ```yaml
    # values.yaml
    pmmEnv:
      PMM_DATA_RETENTION: "720h"  # 30 days (recommended)
    ```

2. Apply the change:

    ```sh
    helm upgrade pmm percona/pmm \
      --namespace monitoring \
      --values values.yaml
    ```

### Configure resource limits

Adjust resources based on your monitoring scale:
```yaml
# values.yaml
pmmResources:
  requests:
    memory: "4Gi"
    cpu: "2"
  limits:
    memory: "8Gi"
    cpu: "4"
```

**Sizing guidelines**

| Monitored databases | Memory | CPU | Storage |
|-------------------|--------|-----|---------|
| 1-10 | 4 GB | 2 cores | 50 GB |
| 11-50 | 8 GB | 4 cores | 100 GB |
| 51-100 | 16 GB | 8 cores | 200 GB |
| 100+ | 32+ GB | 16+ cores | 500+ GB |

## Manage PMM Server

### Connect monitoring clients

To monitor your databases, install PMM Client on each database host and connect it to PMM Server:
{.power-number}

1. [Install PMM Client](../install-pmm/install-pmm-client/index.md) on your database hosts.

2. Get the PMM Server address:
  ```sh
  # For hostname-based load balancers
  export PMM_SERVER=$(kubectl get svc -n monitoring monitoring-service \
    -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

  # Or for IP-based load balancers, uncomment the following:
  # export PMM_SERVER=$(kubectl get svc -n monitoring monitoring-service \
  #   -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  ```

3. Connect to PMM Server and add your database:
  ```sh
  # Configure PMM Client with the server URL
  pmm-admin config --server-url=https://admin:password@${PMM_SERVER}:443

  # Add a database service (example: MySQL)
  pmm-admin add mysql \
    --username=pmm \
    --password=pass \
    --query-source=perfschema \
    --host=mysql-host
  ```

### Test automatic recovery

Verify Kubernetes automatically recovers PMM:

**Test 1: Delete pod**
```sh
# Delete the PMM pod
kubectl delete pod -n monitoring -l app.kubernetes.io/name=pmm

# Watch Kubernetes recreate it
kubectl get pods -n monitoring -w

# Check pod is running on same or different node
kubectl get pod -n monitoring -l app.kubernetes.io/name=pmm -o wide
```

Recovery should complete in 30-60 seconds on the same node.

**Test 2: Drain node (simulate node failure)**
```sh
# Get node running PMM
export PMM_NODE=$(kubectl get pod -n monitoring -l app.kubernetes.io/name=pmm \
  -o jsonpath='{.items[0].spec.nodeName}')

# Drain the node
kubectl drain $PMM_NODE --ignore-daemonsets --delete-emptydir-data

# Watch PMM reschedule to another node
kubectl get pods -n monitoring -w

# Check PMM is running on different node
kubectl get pod -n monitoring -l app.kubernetes.io/name=pmm -o wide

# Uncordon the node when done testing
kubectl uncordon $PMM_NODE
```

Recovery should complete in 2-5 minutes with pod rescheduled to a healthy node.

### Monitor PMM health

Check PMM status and resource usage:
```sh
# Check pod status
kubectl get pods -n monitoring -l app.kubernetes.io/name=pmm

# View pod logs
kubectl logs -n monitoring -l app.kubernetes.io/name=pmm --tail=100

# Check resource usage
kubectl top pod -n monitoring -l app.kubernetes.io/name=pmm

# Check persistent volume
kubectl get pvc -n monitoring

# Describe pod for detailed status
kubectl describe pod -n monitoring -l app.kubernetes.io/name=pmm
```

### Access PMM pod directly

For troubleshooting, access the PMM pod:
```sh
# Execute commands in pod
kubectl exec -it -n monitoring pmm-0 -- bash

# Check PMM Server status
kubectl exec -n monitoring pmm-0 -- pmm-admin status

# View available disk space
kubectl exec -n monitoring pmm-0 -- df -h /srv
```

### Backup and restore

Back up the persistent volume using Kubernetes VolumeSnapshots. To restore, replace the PMM volume with one created from the snapshot. Both stop PMM Server while they run, so the volume is not written to during the snapshot or the restore.

**Create backup**
```sh
SNAPSHOT=pmm-backup-$(date +%Y%m%d)

# Stop PMM Server
kubectl scale statefulset pmm -n monitoring --replicas=0
kubectl wait --for=delete pod/pmm-0 -n monitoring --timeout=120s

# Create VolumeSnapshot (requires CSI driver with snapshot support)
kubectl create -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: $SNAPSHOT
  namespace: monitoring
spec:
  volumeSnapshotClassName: csi-snapclass
  source:
    persistentVolumeClaimName: pmm-storage-pmm-0
EOF

# Wait for the snapshot, then start PMM Server
kubectl wait --for=jsonpath='{.status.readyToUse}'=true volumesnapshot/$SNAPSHOT -n monitoring --timeout=30m
kubectl scale statefulset pmm -n monitoring --replicas=1
```

**Restore from backup**
```sh
# Stop PMM Server
kubectl scale statefulset pmm -n monitoring --replicas=0
kubectl wait --for=delete pod/pmm-0 -n monitoring --timeout=120s

# Delete the current PMM volume. This permanently deletes the data on it.
kubectl delete pvc pmm-storage-pmm-0 -n monitoring

# Recreate the volume from the snapshot, under the name the PMM pod uses
kubectl create -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pmm-storage-pmm-0
  namespace: monitoring
spec:
  dataSource:
    name: pmm-backup-20250119
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi  # At least the size of the original volume
EOF

# Start PMM Server
kubectl scale statefulset pmm -n monitoring --replicas=1
```

### Upgrade PMM

The pod restarts during the upgrade, so expect a brief monitoring interruption (2-5 minutes) while Kubernetes reschedules it.

```sh
# Update Helm repository
helm repo update percona

# Check available versions
helm search repo percona/pmm --versions

# Upgrade to latest version
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml

# Or upgrade to a specific chart version
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml \
  --version <chart-version>

# Monitor upgrade progress
kubectl rollout status statefulset -n monitoring pmm
```


### Rollback upgrade

If an upgrade fails, rollback to the previous version:
```sh
# View release history
helm history pmm -n monitoring

# Rollback to previous version
helm rollback pmm -n monitoring

# Or rollback to specific revision
helm rollback pmm 2 -n monitoring
```

## Get help

- [PMM Community Forums](https://per.co.na/PMM3_forums)
- [Contact Percona Support](https://www.percona.com/services/support)
- [Report bugs or technical issues](https://perconadev.atlassian.net/jira/software/c/projects/PMM/issues/)
- [Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm)
- [Kubernetes best practices for PMM](../install-pmm/install-pmm-server/deployment-options/helm/index.md)

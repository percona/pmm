# Install PMM Single-Instance HA

Deploy PMM Server on Kubernetes using the `percona/pmm` Helm chart. For an overview of this deployment type and when to use it, see [Understand Single-Instance HA](HA-kubernetes-single-instance.md).

## Prerequisites

Storage requirements scale with the number of monitored services and data retention period.

=== "Required software"

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

    - **CPU**: 4+ cores
    - **Memory**: 8+ GB RAM
    - **Storage**: 100+ GB with fast SSD-backed persistent volumes
    - **Multiple nodes**: At least 2 worker nodes for automatic rescheduling


### Verify prerequisites

Check your cluster meets the requirements:
```sh
# Check Kubernetes version
kubectl version --short

# Check available resources
kubectl top nodes

# Verify storage classes
kubectl get storageclass

# Check Helm version
helm version --short
```

## Installation

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

    5. Open `https://<EXTERNAL-IP>` in your browser and log in with default credentials: `admin`/`admin` (change immediately after first login).

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

        resources:
          requests:
            memory: 4Gi
            cpu: 2
          limits:
            memory: 8Gi
            cpu: 4

        # Enable persistent volume
        persistence:
          enabled: true
          storageClass: "fast-ssd"
          size: 100Gi

        # Configure data retention (default: 30 days)
        env:
          - name: PMM_DATA_RETENTION
            value: "720h"  # 30 days

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
kubectl logs -n monitoring -l app=pmm

# Wait for pod to be ready
kubectl wait --for=condition=ready pod \
  -l app=pmm \
  -n monitoring \
  --timeout=300s
```

Expected output:
```
pod/pmm-0   1/1   Running   0   2m
```

## Configuration

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

Choose the service type that fits your environment and apply the configuration.

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

    !!! note "Prerequisites"
        Ensure you have an ingress controller (e.g., NGINX, Traefik) and cert-manager installed in your cluster.
        
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

**Choose appropriate storage class**
```yaml
# values.yaml
persistence:
  storageClass: "fast-ssd"  # Use fastest available
  size: 100Gi
  
  # Optional: Use existing PVC
  # existingClaim: "pmm-data"
  
  # Optional: Storage selector
  # selector:
  #   matchLabels:
  #     type: ssd
```

**Common storage classes by provider**

- **AWS**: `gp3` (recommended), `gp2`, `io1`
- **GCP**: `pd-ssd`, `pd-balanced`
- **Azure**: `managed-premium`, `managed`
- **On-premise**: Check with `kubectl get storageclass`

### Configure data retention

Set how long PMM retains monitoring data:
```yaml
# values.yaml
env:
  - name: PMM_DATA_RETENTION
    value: "720h"  # 30 days (recommended)
```

Common retention periods:

- `168h`: 7 days (minimal storage)
- `720h`: 30 days (recommended)
- `2160h`: 90 days (compliance/audit)

Apply changes:
```sh
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml
```

### Configure resource limits

Adjust resources based on your monitoring scale:
```yaml
# values.yaml
resources:
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

## Operations

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
kubectl delete pod -n monitoring -l app=pmm

# Watch Kubernetes recreate it
kubectl get pods -n monitoring -w

# Check pod is running on same or different node
kubectl get pod -n monitoring -l app=pmm -o wide
```

Recovery should complete in 30-60 seconds on the same node.

**Test 2: Drain node (simulate node failure)**
```sh
# Get node running PMM
export PMM_NODE=$(kubectl get pod -n monitoring -l app=pmm \
  -o jsonpath='{.items[0].spec.nodeName}')

# Drain the node
kubectl drain $PMM_NODE --ignore-daemonsets --delete-emptydir-data

# Watch PMM reschedule to another node
kubectl get pods -n monitoring -w

# Check PMM is running on different node
kubectl get pod -n monitoring -l app=pmm -o wide

# Uncordon the node when done testing
kubectl uncordon $PMM_NODE
```

Recovery should complete in 2-5 minutes with pod rescheduled to a healthy node.

### Monitor PMM health

Check PMM status and resource usage:
```sh
# Check pod status
kubectl get pods -n monitoring -l app=pmm

# View pod logs
kubectl logs -n monitoring -l app=pmm --tail=100

# Check resource usage
kubectl top pod -n monitoring -l app=pmm

# Check persistent volume
kubectl get pvc -n monitoring

# Describe pod for detailed status
kubectl describe pod -n monitoring -l app=pmm
```

### Access PMM pod directly

For troubleshooting, access the PMM pod:
```sh
# Execute commands in pod
kubectl exec -it -n monitoring -l app=pmm -- bash

# Check PMM Server status
kubectl exec -n monitoring -l app=pmm -- pmm-admin status

# View available disk space
kubectl exec -n monitoring -l app=pmm -- df -h /srv
```

### Backup and restore

**Create backup**
```sh
# Create VolumeSnapshot (requires CSI driver with snapshot support)
kubectl create -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: pmm-backup-$(date +%Y%m%d)
  namespace: monitoring
spec:
  volumeSnapshotClassName: csi-snapclass
  source:
    persistentVolumeClaimName: pmm-storage
EOF

# Verify snapshot
kubectl get volumesnapshot -n monitoring
```

**Restore from backup**
```sh
# Create new PVC from snapshot
kubectl create -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pmm-storage-restored
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
      storage: 100Gi
EOF

# Update Helm values to use restored PVC
# values.yaml:
# persistence:
#   existingClaim: pmm-storage-restored

# Upgrade deployment
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml
```

### Upgrade PMM

Perform zero-downtime upgrades:
```sh
# Update Helm repository
helm repo update percona

# Check available versions
helm search repo percona/pmm --versions

# Upgrade to latest version
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml

# Or upgrade to specific version
helm upgrade pmm percona/pmm \
  --namespace monitoring \
  --values values.yaml \
  --version 3.5.0

# Monitor upgrade progress
kubectl rollout status statefulset -n monitoring pmm
```

The StatefulSet ensures zero-downtime rolling updates with health checks.

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

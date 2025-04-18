# Back up PMM Server Helm deployment

Create backups of your PMM Server Kubernetes deployment to protect your monitoring data and configuration.

# Prerequisites

- Running PMM Server deployed with Helm
- Storage class with snapshot support
- Appropriate permissions to create volume snapshots

## Understanding Kubernetes storage for PMM Server

PMM Server Helm chart uses [PersistentVolume and PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to allocate storage in the Kubernetes cluster.

Volumes could be pre-provisioned and dynamic. PMM chart supports both and exposes it through [PMM storage configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-storage-configuration).

Backups for the PMM Server currently support only storage layer backups and thus require:

 - a [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) that supports volume snapshots
 - a [VolumeSnapshotClass](https://kubernetes.io/docs/concepts/storage/volume-snapshot-classes/) configured for your environment

## Verify snapshot support

Before attempting a backup, verify that your cluster supports volume snapshots:

```sh
# Check available storage classes
kubectl get sc

# Check available volume snapshot classes
kubectl get volumesnapshotclass
```

### Storage considerations
Volume snapshot support varies by platform:

- Cloud providers: May incur additional costs for snapshot storage
- On-premises: Requires storage drivers with snapshot capabilities
- Storage capacity: Ensure sufficient space for snapshots

## Create a PMM Server backup

To create a backup of your PMM Server:  
{.power-number}

1. Identify the current PMM version (for restoration purposes):

    ```sh
    kubectl get deployment pmm -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2
    ```

2. Scale down PMM Server to ensure data consistency:

    ```sh
    kubectl scale statefulset pmm --replicas=0
    kubectl wait --for=jsonpath='{.status.replicas}'=0 statefulset pmm
    ```

    ??? example "Expected output"
        ```sh
        statefulset.apps/pmm scaled
        statefulset.apps/pmm condition met
        ```

3. Create a volume snapshot:

    ```sh
    cat <<EOF | kubectl create -f -
    apiVersion: snapshot.storage.k8s.io/v1
    kind: VolumeSnapshot
    metadata:
      name: before-v2.34.0-upgrade
      labels:
        app.kubernetes.io/name: pmm
    spec:
      volumeSnapshotClassName: csi-hostpath-snapclass
      source:
        persistentVolumeClaimName: pmm-storage-pmm-0
    EOF
    ```

    ??? example "Expected output"
        ```sh
        volumesnapshot.snapshot.storage.k8s.io/pmm-backup-20230615 created
        ```

4. Wait for the snapshot to complete:

    ```sh
    kubectl wait --for=jsonpath='{.status.readyToUse}'=true VolumeSnapshot/before-v2.34.0-upgrade
    kubectl scale statefulset pmm --replicas=1
    ```

    ??? example "Expected output"
        ```sh
        volumesnapshot.snapshot.storage.k8s.io/pmm-backup-20230615 condition met
        ```

5. Restart PMM Server:

    ```sh
    kubectl scale statefulset pmm --replicas=1
    ```

6. Verify that PMM Server is running again:

    ```sh
    kubectl get pods -l app.kubernetes.io/name=pmm
    ```

    !!! note "PMM scale"
        Only one replica set is currently supported for PMM Server on Kubernetes.

## List available backups
To view your available PMM Server backups:

  ```sh
  kubectl get volumesnapshot -l app.kubernetes.io/name=pmm
  ```

## Backup rotation
For production environments, implement a backup rotation policy:

  ```sh
  # List backups older than 30 days
  OLD_BACKUPS=$(kubectl get volumesnapshot -l app.kubernetes.io/name=pmm -o jsonpath='{range .items[?(@.metadata.creationTimestamp < "'$(date -d "30 days ago" -Iseconds)'")]}{.metadata.name}{"\n"}{end}')

  # Delete old backups
  for backup in $OLD_BACKUPS; do
    kubectl delete volumesnapshot $backup
  done
  ```

## Next steps

- [Restore PMM Server from backup](restore_container_helm.md)
- [Upgrade PMM Server on Kubernetes](../../../../pmm-upgrade/upgrade_helm.md)
- [Configure advanced storage options](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-storage-configuration)
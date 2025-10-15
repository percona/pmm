# Restore Helm chart

Recover your PMM Server installation from a previously created volume snapshot.

## Prerequisites

- An existing [volume snapshot backup](backup_container_helm.md) of PMM Server
- Access to the Kubernetes cluster where the backup was created
- Helm v3 installed and configured
- Knowledge of the PMM version used in the backup

## Preparing for restoration
Before restoring, gather the necessary information:
{.power-number}

1. List available snapshots to identify the one for restoration:
   ```sh
   kubectl get volumesnapshot -l app.kubernetes.io/name=pmm
   ```

2. Note the snapshot name and creation date:
  ```sh
    NAME                   READYTOUSE   SOURCEPVC           SOURCESNAPSHOTCONTENT   RESTORESIZE   SNAPSHOTCLASS            SNAPSHOTCONTENT                                    CREATIONTIME   AGE
    pmm-backup-20230615    true         pmm-storage-pmm-0                           10Gi          csi-hostpath-snapclass   snapcontent-c9a3d320-be77-49c9-85ff-8257e761f05d   3h36m          3h36m
  ```

3. Verify that the version of PMM Server you plan to restore is compatible with the snapshot (must be equal to or newer than the backup version)

## Restore PMM Server from snapshot

To restore PMM Server from a snapshot:
{.power-number}

1. Remove the existing PMM Server deployment:
    ```sh
    helm uninstall pmm
    ```
2. Wait for resources to be cleaned up:
    ```sh
    kubectl wait --for=delete pod/pmm-0 --timeout=120s
    ```
3. Restore PMM Server using the snapshot as a data source, replacing 3.1.0 with a PMM version that is equal to or newer than the version used in the backup:

    ```sh
    helm install pmm \
    --set image.tag="3.1.0" \
    --set storage.name="pmm-storage-old" \
    --set storage.dataSource.name="before-v3.1.0-upgrade" \
    --set storage.dataSource.kind="VolumeSnapshot" \
    --set storage.dataSource.apiGroup="snapshot.storage.k8s.io" \
    --set secret.create=false \
    --set secret.name=pmm-secret \
    percona/pmm
    ```

4. Verify the restoration:
    ```sh
    kubectl get pods -l app.kubernetes.io/name=pmm
    ```

5. Check that PMM Server is running properly:
    ```sh
    kubectl port-forward svc/pmm-service 443:443
    ```
6. Access PMM Server at `https://localhost:443`. 

## Managing persistent volumes

After restoration, you'll have multiple PVCs in your cluster:  
{.power-number}

1. List the persistent volume claims:

    ```sh
    kubectl get pvc
    ```

    ??? example "Expected output"
        ```sh
        NAME                    STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
        pmm-restored-pmm-0      Bound    pvc-70e5d2eb-570f-4087-9515-edf2f051666d   10Gi       RWO            csi-hostpath-sc   3s
        pmm-storage-pmm-0       Bound    pvc-9dbd9160-e4c5-47a7-bd90-bff36fc1463e   10Gi       RWO            csi-hostpath-sc   89m
        ```

2. List the underlying persistent volumes:

    ```sh
    kubectl get pv
    ```

3. Clean up old volumes when they're no longer needed:

    ```sh
    # Only delete these when you've confirmed the restoration is successful
    kubectl delete pvc pmm-storage-pmm-0
    kubectl delete pv <corresponding-pv-name>
    ```

## Next steps

- [Verify monitoring data](../../../../use/dashboard-inventory.md) in the restored PMM Server
- [Configure backup schedule](backup_container_helm.md) for your restored environment
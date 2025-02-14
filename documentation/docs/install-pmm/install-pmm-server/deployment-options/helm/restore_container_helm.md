# Restore Helm chart

The version of the PMM Server should be greater than or equal to the version in a snapshot. To restore from the snapshot, delete the old deployment first:
```sh
helm uninstall pmm
```

And then use snapshot configuration to start the PMM Server again with the correct version and correct storage configuration:
```sh
helm install pmm \
--set image.tag="2.34.0" \
--set storage.name="pmm-storage-old" \
--set storage.dataSource.name="before-v2.34.0-upgrade" \
--set storage.dataSource.kind="VolumeSnapshot" \
--set storage.dataSource.apiGroup="snapshot.storage.k8s.io" \
--set secret.create=false \
--set secret.name=pmm-secret \
percona/pmm
```

Here, we created a new `pmm-storage-old` PVC with data from the snapshot. So, there are a couple of PV and PVCs available in a cluster.

```
$ kubectl get pvc
NAME                    STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS      AGE
pmm-storage-old-pmm-0   Bound    pvc-70e5d2eb-570f-4087-9515-edf2f051666d   10Gi       RWO            csi-hostpath-sc   3s
pmm-storage-pmm-0       Bound    pvc-9dbd9160-e4c5-47a7-bd90-bff36fc1463e   10Gi       RWO            csi-hostpath-sc   89m

$ kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                           STORAGECLASS      REASON   AGE
pvc-70e5d2eb-570f-4087-9515-edf2f051666d   10Gi       RWO            Delete           Bound    default/pmm-storage-old-pmm-0   csi-hostpath-sc            4m50s
pvc-9dbd9160-e4c5-47a7-bd90-bff36fc1463e   10Gi       RWO            Delete           Bound    default/pmm-storage-pmm-0       csi-hostpath-sc            93m
```

Delete unneeded PVC when you are sure you don't need them.



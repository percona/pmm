# Back up Helm charts

PMM helm chart uses [PersistentVolume and PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to allocate storage in the Kubernetes cluster.

Volumes could be pre-provisioned and dynamic. PMM chart supports both and exposes it through [PMM storage configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-storage-configuration).

Backups for the PMM Server currently support only storage layer backups and thus require [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) and [VolumeSnapshotClass](https://kubernetes.io/docs/concepts/storage/volume-snapshot-classes/).

Validate the correct configuration by using these commands:
```sh
kubectl get sc
kubectl get volumesnapshotclass
```

!!! note alert alert-primary "Storage"
    Storage configuration is Hardware and Cloud specific. There could be additional costs associated with Volume Snapshots. Check the documentation for your Cloud or for your Kubernetes cluster.

Before taking a [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/), stop the PMM Server. In this step, we will stop PMM (scale to 0 pods), take a snapshot, wait until the snapshot completes, then start PMM Server (scale to 1 pod):
```sh
kubectl scale statefulset pmm --replicas=0
kubectl wait --for=jsonpath='{.status.replicas}'=0 statefulset pmm

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

kubectl wait --for=jsonpath='{.status.readyToUse}'=true VolumeSnapshot/before-v2.34.0-upgrade
kubectl scale statefulset pmm --replicas=1
```

Output:

```
statefulset.apps/pmm scaled
statefulset.apps/pmm condition met
volumesnapshot.snapshot.storage.k8s.io/before-v2.34.0-upgrade created
volumesnapshot.snapshot.storage.k8s.io/before-v2.34.0-upgrade condition met
statefulset.apps/pmm scaled
```

!!! note alert alert-primary "PMM scale"
    Only one replica set is currently supported.

You can view available snapshots by executing the following command:
```sh
kubectl get volumesnapshot
```



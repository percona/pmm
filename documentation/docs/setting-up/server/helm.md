# Helm

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. Percona Helm charts can be found in [percona/percona-helm-charts](https://github.com/percona/percona-helm-charts) repository on Github.

## Before you start

- Install Helm following its [official installation instructions](https://docs.helm.sh/using_helm/#installing-helm).
- Kubernetes cluster that [Helm supports](https://helm.sh/docs/topics/kubernetes_distros/)

!!! note alert alert-primary ""
    Helm v3 is needed to run the following steps.

Refer to [Kubernetes Supported versions](https://kubernetes.io/releases/version-skew-policy/#supported-versions) and [Helm Version Support Policy](https://helm.sh/docs/topics/version_skew/) to find the supported versions.

PMM should be platform-agnostic, but it requires escalated privileges inside a container. It is necessary to have a `root` user inside the PMM container. Thus, PMM would not work for Kubernetes Platforms such as OpenShift or others that have hardened Security Context Constraints, for example:

- [Security context constraints (SCCs)
](https://docs.openshift.com/container-platform/latest/security/container_security/security-platform.html#security-deployment-sccs_security-platform)
- [Managing security context constraints](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)

Kubernetes platforms offer a different set of capabilities. To use PMM in production, you would need backups and, thus storage driver that supports snapshots. Consult your provider for Kubernetes and Cloud storage capabilities.

## Locality and Availability

You should not run the PMM monitoring server along with the monitored database clusters and services on the same system.

Please ensure proper locality either by physically separating workloads in Kubernetes clusters or running separate Kubernetes clusters for the databases and monitoring workloads.

You can physically separate workloads by properly configuring Kubernetes nodes, affinity rules, label selections, etc.

Also, ensure that the Kubernetes cluster has [high availability](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) so that in case of a node failure, the monitoring service will be running and capturing the required data.

## Use Helm to install PMM server on Kubernetes clusters

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.


!!! summary alert alert-info "Summary"
    - Setup PMM admin password
    - Install
    - Configuration parameters
    - PMM environment variables
    - PMM SSL certificates
    - Backup
    - Upgrade
    - Restore
    - Uninstall

---

### Setup PMM admin password

Create Kubernetes secret with PMM admin password:
```sh
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Secret
metadata:
  name: pmm-secret
  labels:
    app.kubernetes.io/name: pmm
type: Opaque
data:
# base64 encoded password
# encode some password: `echo -n "admin" | base64`
  PMM_ADMIN_PASSWORD: YWRtaW4=
EOF
```

To get admin password execute:

```sh
kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
```

### Install

To install the chart with the release name `pmm`:

```sh
helm repo add percona https://percona.github.io/percona-helm-charts/
helm install pmm \
--set secret.create=false \
--set secret.name=pmm-secret \
percona/pmm
```
The command deploys PMM on the Kubernetes cluster in the default configuration and specified secret. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

<div hidden>
```sh
helm uninstall pmm
```
</div>

!!! hint alert alert-success "Tip"
    List all releases using `helm list`.

### Parameters

The list of Parameters is subject to change from release to release. Check the [Parameters](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters) section of the PMM Helm Chart.

!!! hint alert alert-success "Tip"
    You can list the default parameters [values.yaml](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) or get them from chart definition: `helm show values percona/pmm`

Specify each parameter using the `--set key=value[,key=value]` or `--set-string key=value[,key=value]` arguments to `helm install`. For example,

```sh
helm install pmm \
--set secret.create=false --set secret.name=pmm-secret \
--set-string pmmEnv.DISABLE_UPDATES="1" \
--set service.type="NodePort" \
--set storage.storageClassName="linode-block-storage-retain" \
    percona/pmm
```

The command above installs PMM, configuring the service network type as `NodePort` and setting the storage class to `linode-block-storage-retain` for persistent storage on LKE.```

<div hidden>
```sh
helm uninstall pmm
```
</div>

!!! caution alert alert-warning "Important"
    Once this chart is deployed, it is impossible to change the application's access credentials, such as password, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools (if available)

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example:

```sh
helm show values percona/pmm > values.yaml

#change needed parameters in values.yaml, you need `yq` tool pre-installed
yq -i e '.secret.create |= false' values.yaml

helm install pmm -f values.yaml percona/pmm
```

### [PMM environment variables](docker.md#environment-variables)

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `pmmEnv` property.

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
```

### PMM SSL certificates

PMM ships with self signed SSL certificates to provide secure connection between client and server ([check here](../../how-to/secure.md#ssl-encryption)).

You will see the warning when connecting to PMM. To further increase security, you should provide your certificates and add values of credentials to the fields of the `cert` section:

```yaml
certs:
  name: pmm-certs
  files:
    certificate.crt: <content>
    certificate.key: <content>
    ca-certs.pem: <content>
    dhparam.pem: <content>
```

Another approach to set up TLS certificates is to use the Ingress controller, see [TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls). PMM helm chart supports Ingress. See [PMM network configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-network-configuration).

## Backup

PMM helm chart uses [PersistentVolume and PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to allocate storage in the Kubernetes cluster.

Volumes could be pre-provisioned and dynamic. PMM chart supports both and exposes it through [PMM storage configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-storage-configuration).

Backups for the PMM server currently support only storage layer backups and thus require [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) and [VolumeSnapshotClass](https://kubernetes.io/docs/concepts/storage/volume-snapshot-classes/).

Validate the correct configuration by using these commands:
```sh
kubectl get sc
kubectl get volumesnapshotclass
```

!!! note alert alert-primary "Storage"
    Storage configuration is Hardware and Cloud specific. There could be additional costs associated with Volume Snapshots. Check the documentation for your Cloud or for your Kubernetes cluster.

Before taking a [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/), stop the PMM server. In this step, we will stop PMM (scale to 0 pods), take a snapshot, wait until the snapshot completes, then start PMM server (scale to 1 pod):
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

### Upgrades

Percona will release a new chart updating its containers if a new version of the main container is available, there are any significant changes, or critical vulnerabilities exist.

By default the UI update feature is disabled and should not be enabled. Do not modify that parameter when customizing the `values.yaml` file:

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
```

Before updating the helm chart, it is recommended to pre-pull the image on the node where PMM is running, as the PMM images could be large and could take time to download.

Update PMM as follows:

```sh
helm repo update percona
helm upgrade pmm -f values.yaml percona/pmm
```

This will check updates in the repo and upgrade deployment if the updates are available.

## Restore

The version of the PMM server should be greater than or equal to the version in a snapshot. To restore from the snapshot, delete the old deployment first:
```sh
helm uninstall pmm
```

And then use snapshot configuration to start the PMM server again with the correct version and correct storage configuration:
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

### Uninstall

To uninstall `pmm` deployment:

```sh
helm uninstall pmm
```

This command takes a release name and uninstalls the release.

It removes all resources associated with the last release of the chart as well as the release history.

Helm will not delete PVC, PV, and any snapshots. Those need to be deleted manually.

Also, delete PMM `Secret` if no longer required:
```sh
kubectl delete secret pmm-secret
```

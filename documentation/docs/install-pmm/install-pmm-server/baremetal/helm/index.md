# Install PMM Server with Helm on the Kubernetes clusters


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

## Install PMM Server

??? info "Summary"

    !!! summary alert alert-info ""
        - Setup pmm-admin password
        - Install
        - Configuration parameters
        - PMM environment variables
        - PMM SSL certificates
        - Backup
        - Upgrade
        - Restore
        - Uninstall

    ---

### Set up pmm-admin password

Create Kubernetes secret with pmm-admin password:
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
--set service.type="NodePort" \
--set storage.storageClassName="linode-block-storage-retain" \
    percona/pmm
```

The above command installs PMM and sets the Service network type to `NodePort` and storage class to `linode-block-storage-retain` for persistence storage on LKE.

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

### [PMM environment variables](../docker/env_var.md)

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `pmmEnv` property.

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
```

### PMM SSL certificates

PMM ships with self signed SSL certificates to provide secure connection between client and server ([check here](../../../../pmm-admin/security/ssl_encryption.md)).

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









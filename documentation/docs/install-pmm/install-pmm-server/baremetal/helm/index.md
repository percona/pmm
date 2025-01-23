@ -1,153 +1,158 @@
# Install PMM Server with Helm on Kubernetes clusters

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. You can find Percona Helm charts in [our GitHub repository](https://github.com/percona/percona-helm-charts). 

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. Percona Helm charts can be found in [percona/percona-helm-charts](https://github.com/percona/percona-helm-charts) repository on Github.
## Prerequisites

## Before you start
  - [Helm v3](https://docs.helm.sh/using_helm/#installing-helm)
  - Helm chart 1.4.0+
  - Supported cluster according to [Supported Kubernetes](https://kubernetes.io/releases/version-skew-policy/#supported-versions) and [Supported Helm](https://helm.sh/docs/topics/version_skew/) versions
  - Storage driver with snapshot support (for backups)

- Install Helm following its [official installation instructions](https://docs.helm.sh/using_helm/#installing-helm).
- Kubernetes cluster that [Helm supports](https://helm.sh/docs/topics/kubernetes_distros/)
## Platform limitations

!!! note alert alert-primary ""
    Helm v3 is needed to run the following steps.
PMM is platform-agnostic but requires `root` privileges inside containers. Due to this requirement, PMM is incompatible with:

Refer to [Kubernetes Supported versions](https://kubernetes.io/releases/version-skew-policy/#supported-versions) and [Helm Version Support Policy](https://helm.sh/docs/topics/version_skew/) to find the supported versions.
- Platforms with [Security context constraints (SCCs)](https://docs.openshift.com/container-platform/latest/security/container_security/security-platform.html#security-deployment-sccs_security-platform), like OpenShift
- Platforms with [restrictive security context management](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)

PMM should be platform-agnostic, but it requires escalated privileges inside a container. It is necessary to have a `root` user inside the PMM container. Thus, PMM would not work for Kubernetes Platforms such as OpenShift or others that have hardened Security Context Constraints, for example:
## Storage requirements

- [Security context constraints (SCCs)
](https://docs.openshift.com/container-platform/latest/security/container_security/security-platform.html#security-deployment-sccs_security-platform)
- [Managing security context constraints](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)
Different Kubernetes platforms offer varying capabilities. To use PMM in production: 

Kubernetes platforms offer a different set of capabilities. To use PMM in production, you would need backups and, thus storage driver that supports snapshots. Consult your provider for Kubernetes and Cloud storage capabilities.
- ensure your platform provides storage drivers supporting snapshots for backups
- consult your provider about Kubernetes and Cloud storage capabilities

## Locality and Availability

You should not run the PMM monitoring server along with the monitored database clusters and services on the same system.
## Deployment best practices

Please ensure proper locality either by physically separating workloads in Kubernetes clusters or running separate Kubernetes clusters for the databases and monitoring workloads.
For optimal monitoring:
{.power-number}

You can physically separate workloads by properly configuring Kubernetes nodes, affinity rules, label selections, etc.
1. Separate PMM Server from monitored systems by either:

Also, ensure that the Kubernetes cluster has [high availability](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) so that in case of a node failure, the monitoring service will be running and capturing the required data.
  - using separate Kubernetes clusters for monitoring and databases
  - configuring workload separation through node configurations, affinity rules, and label selectors

## Install PMM Server
2. Enable [high availability](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) to ensure continuous monitoring during node failures

??? info "Summary"
## Installation PMM Server on your Kubernetes cluster

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
Create the required Kubernetes secret and deploy PMM Server using Helm:

    ---
1. Create Kubernetes secret to set up `pmm-admin` password:
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

### Set up pmm-admin password
2. Get admin password:

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
  ```sh
  kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
  ```

To get admin password execute:
3. Add the Percona repository and deploy PMM Server with default settings and your secret. See configuration parameters for customization. See [configuration parameters]((#view-available-parameters)) for customization.

```sh
kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
```

### Install
  ```sh
  helm repo add percona https://percona.github.io/percona-helm-charts/
  helm install pmm \
  --set secret.create=false \
  --set secret.name=pmm-secret \
  percona/pmm
  ```

To install the chart with the release name `pmm`:
4. Vrify the deployment, listing all releases: `helm list`.

```sh
helm repo add percona https://percona.github.io/percona-helm-charts/
helm install pmm \
--set secret.create=false \
--set secret.name=pmm-secret \
percona/pmm
```
The command deploys PMM on the Kubernetes cluster in the default configuration and specified secret. The [Parameters](#parameters) section lists the parameters that can be configured during installation.
### Configure PMM Server

<div hidden>
```sh
helm uninstall pmm
```
</div>
#### View available parameters

!!! hint alert alert-success "Tip"
    List all releases using `helm list`.
Check the list of available parameters in the [PMM Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters). You can also list the default parameters by either: 
-  check [values.yaml file](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) in our repository
- run the chart definition: `helm show values percona/pmm`

### Parameters
#### Set configuration values

The list of Parameters is subject to change from release to release. Check the [Parameters](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters) section of the PMM Helm Chart.
Configure PMM Server using either command-line arguments or a YAML file:

!!! hint alert alert-success "Tip"
    You can list the default parameters [values.yaml](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) or get them from chart definition: `helm show values percona/pmm`
 - using command-line arguments: 
    ```sh
    helm install pmm \
    --set secret.create=false --set secret.name=pmm-secret \
    --set service.type="NodePort" \
    --set storage.storageClassName="linode-block-storage-retain" \
        percona/pmm
    ```
- using a .yaml configuration file: 

Specify each parameter using the `--set key=value[,key=value]` or `--set-string key=value[,key=value]` arguments to `helm install`. For example,

```sh
helm install pmm \
--set secret.create=false --set secret.name=pmm-secret \
--set service.type="NodePort" \
--set storage.storageClassName="linode-block-storage-retain" \
    percona/pmm
```
### Configure PMM Server

The above command installs PMM and sets the Service network type to `NodePort` and storage class to `linode-block-storage-retain` for persistence storage on LKE.
#### View available parameters

<div hidden>
```sh
helm uninstall pmm
```
</div>
Check the list of available parameters in the [PMM Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters). You can also list the default parameters by either: 
-  check [values.yaml file](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) in our repository
- run the chart definition: `helm show values percona/pmm`

!!! caution alert alert-warning "Important"
    Once this chart is deployed, it is impossible to change the application's access credentials, such as password, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools (if available)
#### Set configuration values

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example:
Configure PMM Server using either:

```sh
helm show values percona/pmm > values.yaml
- command-line arguments:

#change needed parameters in values.yaml, you need `yq` tool pre-installed
yq -i e '.secret.create |= false' values.yaml
  ```sh
  helm install pmm \
  --set secret.create=false --set secret.name=pmm-secret \
  --set service.type="NodePort" \
  --set storage.storageClassName="linode-block-storage-retain" \
  percona/pmm
  ```

helm install pmm -f values.yaml percona/pmm
```
- .yaml configuration file:
  ```sh
  helm show values percona/pmm > values.yaml
  ``` 
 
#### Change credentials

### [PMM environment variables](../docker/env_var.md)
!!! caution alert alert-warning "Important"
Helm cannot modify application credentials after deployment.

Credential changes after deployment require either:
- redeploying PMM Server with new persistent volumes
- using PMM's built-in administrative tools


### PMM environment variables

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `pmmEnv` property.
Add [environment variables](../docker/env_var.md) for advanced operations (like custom init scripts) using the `pmmEnv` property:

```yaml
pmmEnv:
  PMM_ENABLE_UPDATES: "1"
```

### PMM SSL certificates
### SSL certificates

PMM ships with self signed SSL certificates to provide secure connection between client and server ([check here](../../../../pmm-admin/security/ssl_encryption.md)).
PMM comes with [self-signed SSL certificates]((../../../../pmm-admin/security/ssl_encryption.md)), ensuring a secure connection between the client and server. However, since these certificates are not issued by a trusted authority, you may encounter a security warning when connecting to PMM.

You will see the warning when connecting to PMM. To further increase security, you should provide your certificates and add values of credentials to the fields of the `cert` section:
To enhance security, you have two options: 
{.power-number}

1. Configure custom certificates:

```yaml
certs:
@ -158,9 +163,7 @@ certs:
    ca-certs.pem: <content>
    dhparam.pem: <content>
```

Another approach to set up TLS certificates is to use the Ingress controller, see [TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls). PMM helm chart supports Ingress. See [PMM network configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-network-configuration).

2. Use [Ingress controller with TLS]((https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)) See [PMM network configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-network-configuration) for details.




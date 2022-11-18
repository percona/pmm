# Helm

!!! caution alert alert-warning "Caution"
    PMM on Kubernetes with Helm is currently in [technical preview](../../details/glossary.md#technical-preview) and is subject to change.

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. Percona Helm charts can be found in [percona/percona-helm-charts](https://github.com/percona/percona-helm-charts) repository on Github.

## Before you start

- Install Helm following its [official installation instructions](https://docs.helm.sh/using_helm/#installing-helm).
- Kubernetes cluster that [Helm supports](https://helm.sh/docs/topics/kubernetes_distros/)

!!! note alert alert-primary ""
    Helm v3 is needed to run the following steps.

## Use Helm to install PMM server on Kubernetes clusters

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.


!!! summary alert alert-info "Summary"
    - Install
    - Configuration parameters
    - PMM admin password
    - PMM environment variables
    - PMM SSL certificates
    - Upgrade
    - Uninstall

---

### Install

To install the chart with the release name `pmm`:

```sh
helm repo add percona https://percona.github.io/percona-helm-charts/
helm install pmm percona/pmm
```
The command deploys PMM on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

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
--set-string pmmEnv.ENABLE_DBAAS="1" \
--set service.type="NodePort" \
--set storage.storageClassName="linode-block-storage-retain" \
    percona/pmm
```

The above command installs PMM with the enabled PMM DBaaS feature. Additionally, it sets the Service network type to `NodePort` and storage class to `linode-block-storage-retain` for persistence storage on LKE.

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
#change needed parameters in values.yaml
helm install pmm -f values.yaml percona/pmm
```

### PMM admin password

PMM admin password would be set only on the first deployment. That setting is ignored if PMM was already provisioned and just restarted and/or updated.

If PMM admin password is not set explicitly (default), it will be generated.

To get admin password execute:

```sh
kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
```

### [PMM environment variables](docker.md#environment-variables)

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `pmmEnv` property.

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
  ENABLE_DBAAS: "1"
```

### PMM SSL certificates

PMM ships with self signed SSL certificates to provide secure connection between client and server ([check here](../../how-to/secure.md#ssl-encryption)).
You will see the warning when connecting to PMM. To further increase security, you could provide your certificates and add values of credentials to the fields of the `cert` section:

```yaml
certs:
  name: pmm-certs
  files:
    certificate.crt: <content>
    certificate.key: <content>
    ca-certs.pem: <content>
    dhparam.pem: <content>
```

### Upgrades


Percona will release a new chart updating its containers if a new version of the main container is available, there are any significant changes, or critical vulnerabilities exist.

By default UI update feature is disabled and should not be enabled. Do not modify that parameter or add it while modifying the custom `values.yaml` file:

```yaml
pmmEnv:
  DISABLE_UPDATES: "1"
```

Before updating the helm chart,  it is recommended to pre-pull the image on the node where PMM is running, as the PMM images could be large and could take time to download.

Update PMM as follows:

```sh
helm repo update percona
helm upgrade pmm -f values.yaml percona/pmm
```

This will check updates in the repo and upgrade deployment if the updates are available.

### Uninstall

To uninstall `pmm` deployment:

```sh
helm uninstall pmm
```

This command takes a release name and uninstalls the release.

It removes all of the resources associated with the last release of the chart as well as the release history.

# Install PMM Server with Helm on Kubernetes clusters

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. You can find Percona Helm charts in [our GitHub repository](https://github.com/percona/percona-helm-charts). 

## Prerequisites

  - [Helm v3](https://docs.helm.sh/using_helm/#installing-helm)
  - Helm chart 1.4.0+
  - Supported cluster according to [Supported Kubernetes](https://kubernetes.io/releases/version-skew-policy/#supported-versions) and [Supported Helm](https://helm.sh/docs/topics/version_skew/) versions
  - Storage driver with snapshot support (for backups)

## Platform limitations

PMM is platform-agnostic but requires `root` privileges inside containers. Due to this requirement, PMM is incompatible with:

- platforms with [Security context constraints (SCCs)](https://docs.openshift.com/container-platform/latest/security/container_security/security-platform.html#security-deployment-sccs_security-platform), like OpenShift
- platforms with [restrictive security context management](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)

## Storage requirements

Different Kubernetes platforms offer varying capabilities. 

To use PMM in production: 

- ensure your platform provides storage drivers supporting snapshots for backups
- consult your provider about Kubernetes and Cloud storage capabilities


## Deployment best practices

For optimal monitoring:
{.power-number}

1. Separate PMM Server from monitored systems by either:

    - using separate Kubernetes clusters for monitoring and databases
    - configuring workload separation through node configurations, affinity rules, and label selectors

2. Enable [high availability](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) to ensure continuous monitoring during node failures

## Installation PMM Server on your Kubernetes cluster

Create the required Kubernetes secret and deploy PMM Server using Helm:
{.power-number}

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

2. Get admin password:

    ```sh
    kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
    ```

3. Add the Percona repository and deploy PMM Server with default settings and your secret. See configuration parameters for customization. See [configuration parameters]((#view-available-parameters)) for customization.

    ```sh
    helm repo add percona https://percona.github.io/percona-helm-charts/
    helm install pmm \
    --set secret.create=false \
    --set secret.name=pmm-secret \
    percona/pmm
    ```

4. Verify the deployment, listing all releases: `helm list`.

### Configure PMM Server

#### View available parameters

Check the list of available parameters in the [PMM Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters). You can also list the default parameters by either: 
-  check [values.yaml file](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) in our repository
- run the chart definition: `helm show values percona/pmm`

#### Set configuration values

Configure PMM Server using either command-line arguments or a YAML file:

 - using command-line arguments: 
    ```sh
    helm install pmm \
    --set secret.create=false --set secret.name=pmm-secret \
    --set service.type="NodePort" \
    --set storage.storageClassName="linode-block-storage-retain" \
        percona/pmm
    ```
- using a .yaml configuration file: 


### Configure PMM Server

#### View available parameters

Check the list of available parameters in the [PMM Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters). You can also list the default parameters by either: 
-  check [values.yaml file](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) in our repository
- run the chart definition: `helm show values percona/pmm`

#### Set configuration values

Configure PMM Server using either:

- command-line arguments:

    ```sh
    helm install pmm \
    --set secret.create=false --set secret.name=pmm-secret \
    --set service.type="NodePort" \
    --set storage.storageClassName="linode-block-storage-retain" \
    percona/pmm
    ```

- .yaml configuration file:
  ```sh
  helm show values percona/pmm > values.yaml
  ``` 
 
#### Change credentials

!!! caution alert alert-warning "Important"
    Helm cannot modify application credentials after deployment.

Credential changes after deployment require either:

- redeploying PMM Server with new persistent volumes
- using PMM's built-in administrative tools


### PMM environment variables

Add [environment variables](../docker/env_var.md) for advanced operations (like custom init scripts) using the `pmmEnv` property:

```yaml
pmmEnv:
  PMM_ENABLE_UPDATES: "1"
```

### SSL certificates

PMM comes with [self-signed SSL certificates]((../../../../pmm-admin/security/ssl_encryption.md)), ensuring a secure connection between the client and server. However, since these certificates are not issued by a trusted authority, you may encounter a security warning when connecting to PMM.

To enhance security, you have two options: 
{.power-number}

1. Configure custom certificates:

    ```yaml
    certs:
      name: pmm-certs
      files:
        certificate.crt: <content>
        certificate.key: <content>
        ca-certs.pem: <content>
        dhparam.pem: <content>
    ```

2. Use [Ingress controller with TLS]((https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)) See [PMM network configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-network-configuration) for details.








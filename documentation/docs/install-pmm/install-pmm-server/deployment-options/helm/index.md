# Install PMM Server with Helm on Kubernetes clusters

Deploy PMM Server on Kubernetes using Helm for scalable, orchestrated monitoring in containerized environments.

[Helm](https://github.com/helm/helm) is the package manager for Kubernetes. You can find Percona Helm charts in [our GitHub repository](https://github.com/percona/percona-helm-charts). 

## Prerequisites

  - [Helm v3](https://docs.helm.sh/using_helm/#installing-helm)
  - Kubernetes cluster running a [supported version](https://kubernetes.io/releases/version-skew-policy/#supported-versions) and [Supported Helm](https://helm.sh/docs/topics/version_skew/) versions
  - Storage driver with snapshot support (for backups)
  - `kubectl` configured to communicate with your cluster

## Storage requirements

Different Kubernetes platforms offer varying capabilities: 

- for **production use**, ensure your platform provides storage drivers supporting snapshots for backups
- for **cloud environments**, verify your provider's Kubernetes storage options and costs
- for **on-premises deployments**, confirm your storage solution is compatible with dynamic provisioning

## Deployment best practices

For optimal monitoring in production environments:
{.power-number}

1. Separate PMM Server from monitored systems by either:

    - using separate Kubernetes clusters for monitoring and databases
    - configuring workload separation through node configurations, affinity rules, and label selectors

2. Enable [high availability](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/) to ensure continuous monitoring during node failures

## Install PMM Server on your Kubernetes cluster

Create the required Kubernetes secret and deploy PMM Server using Helm:
{.power-number}

1. Create Kubernetes secret to set up `pmm-admin` password:
    ```bash
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

2. Verify the secret was created and retrieve the password if needed:

    ```bash
    kubectl get secret pmm-secret -o jsonpath='{.data.PMM_ADMIN_PASSWORD}' | base64 --decode
    ```

3. Add the Percona repository and check available PMM versions:

    ```bash
    helm repo add percona [https://percona.github.io/percona-helm-charts/](https://percona.github.io/percona-helm-charts/)
    helm repo update
    ```

4. Choose your PMM version by checking available chart versions:

    ```bash
    helm search repo percona/pmm --versions
    ```

    ??? info "Example output"
        ```text
        NAME        CHART VERSION   APP VERSION DESCRIPTION
        percona/pmm 1.4.3           3.1.0       A Helm chart for Percona Monitoring and Managem...
        percona/pmm 1.4.2           3.1.0       A Helm chart for Percona Monitoring and Managem...
        percona/pmm 1.4.1           3.0.0       A Helm chart for Percona Monitoring and Managem...
        percona/pmm 1.4.0           3.0.0       A Helm chart for Percona Monitoring and Managem...
        percona/pmm 1.3.21          2.44.0      A Helm chart for Percona Monitoring and Managem...
        ```

5. Deploy PMM Server with your chosen version and secret:

    ```bash
    # Choose a specific chart version from the list in previous step
    helm install pmm \
    --set secret.create=false \
    --set secret.name=pmm-secret \
    --version 1.4.3 \
    percona/pmm
    ```

6. Verify the deployment:
    ```bash
    helm list
    kubectl get pods -l app.kubernetes.io/name=pmm
    ```

7. Access PMM Server:

    ```bash
    # If using ClusterIP (default)
    kubectl port-forward svc/pmm-service 443:443

    # If using NodePort
    kubectl get svc pmm-service -o jsonpath='{.spec.ports[0].nodePort}'
    ```
  
### Configure PMM Server

#### View available parameters

Check the list of available parameters in the [PMM Helm chart documentation](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#parameters). You can also list the default parameters by either: 

- check [values.yaml file](https://github.com/percona/percona-helm-charts/blob/main/charts/pmm/values.yaml) in our repository
- run the chart definition: `helm show values percona/pmm`

#### Set configuration values

Configure PMM Server using either command-line arguments or a YAML file:

 - using command-line arguments: 
    ```sh
    helm install pmm \
    --set secret.create=false --set secret.name=pmm-secret \
    --set service.type="NodePort" \
        percona/pmm
    ```
- using a .yaml configuration file: 
  ```sh
  helm show values percona/pmm > values.yaml
  ``` 
 
#### Change credentials

Helm cannot modify application credentials after deployment.  To change credentials after deployment, either:

- redeploy PMM Server with new persistent volumes
- use PMM's built-in administrative tools

### PMM environment variables

Add [environment variables](../docker/env_var.md) for advanced operations (like custom init scripts) using the `pmmEnv` property:

```yaml
pmmEnv:
PMM_ENABLE_UPDATES: "1"
```

### SSL certificates

PMM comes with [self-signed SSL certificates](../../../../admin/security/ssl_encryption.md), ensuring a secure connection between the Client and Server. However, since these certificates are not issued by a trusted authority, you may encounter a security warning when connecting to PMM.

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

2. Use [Ingress controller with TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls). See [PMM network configuration](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#pmm-network-configuration) for details.

## Next steps

- [Register PMM Clients](../../../register-client-node/index.md) with your PMM Server
- [Back up PMM Server Helm deployment](backup_container_helm.md)
- [Configure advanced Kubernetes settings](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm#advanced-configuration)




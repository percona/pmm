# Upgrade PMM Server on a Kubernetes single instance

Upgrade a single PMM Server instance deployed on Kubernetes with the [`percona/pmm`](https://github.com/percona/percona-helm-charts/tree/main/charts/pmm) chart when you want to move to a new PMM release. 

Percona releases new chart versions to update containers when:

- A new version of the main container is available
- Significant changes are made
- Critical vulnerabilities are addressed

## Before you begin

[Create a backup](../install-pmm/install-pmm-server/deployment-options/helm/backup_container_helm.md) before upgrading. Downgrades are not supported, so a backup is your only recovery option if something goes wrong.

## Upgrade PMM Server

Follow these steps to upgrade your PMM Server while preserving your monitoring data and settings. You can restore from your backup if needed.
{.power-number}

1. Update the Helm repository:

    ```sh
    helm repo update percona
    ```

2. Upgrade PMM:

    ```sh
    helm upgrade pmm -f values.yaml --set podSecurityContext.runAsGroup=null --set podSecurityContext.fsGroup=null percona/pmm
    ```

3. After the upgrade, verify that PMM Server is running correctly and all your data is accessible:

    ```sh
    kubectl get pods | grep pmm-server
    ```

4. Check the logs for any errors:

    ```sh
    kubectl logs deployment/pmm-server
    ```

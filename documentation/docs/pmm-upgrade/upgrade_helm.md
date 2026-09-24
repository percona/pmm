# Upgrade PMM Server using Helm

!!! info "Running PMM HA Cluster?"
    This topic covers the single-instance `percona/pmm` chart (one PMM Server pod). If you deployed [PMM HA Cluster](../install-pmm/HA-clustered.md) with the `percona/pmm-ha` chart, use [Upgrade PMM HA Cluster using Helm](upgrade_helm_ha.md) instead—the release name, chart, and rolling-update behavior are different.

Percona releases new chart versions to update containers when:

- A new version of the main container is available
- Significant changes are made
- Critical vulnerabilities are addressed

!!! caution alert alert-warning "UI Update feature disabled by default"
    The UI update feature is disabled by default and should remain so. Do not modify or add the following parameter in your custom `values.yaml` file:
    ```yaml
    pmmEnv:
    PMM_ENABLE_UPDATES: 'false'
    ```

## Before you begin

Before starting the upgrade, complete these preparation steps to ensure you can recover your system if needed and confirm compatibility with the new version:
{.power-number}

1. [Create a backup](../install-pmm/install-pmm-server/deployment-options/helm/backup_container_helm.md) before upgrading, as downgrades are not possible. Therefore, reverting to a previous version requires a backup made prior to the upgrade.

2. To reduce downtime, pre-pull the new image on the node where PMM is running:

    ```sh
    # Replace <version> with the latest PMM version
    docker pull percona/pmm-server:3
    ```

## Upgrade steps

Follow these steps to upgrade your PMM Server while preserving your monitoring data and settings—you can restore from your backup if needed.
{.power-number}

1. Update Helm repository:
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

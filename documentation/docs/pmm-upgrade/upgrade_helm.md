# Upgrade PMM Server using Helm

Percona releases new chart versions to update containers when:

- A new version of the main container is available
- Significant changes are made
- Critical vulnerabilities are addressed

!!! caution alert alert-warning "UI Update feature disabled by default"
    The UI update feature is disabled by default and should remain so. Do not modify or add the following parameter in your custom `values.yaml` file:
    ```yaml
    pmmEnv:
    DISABLE_UPDATES: "1"
    ```

## Before you begin

Before starting the upgrade, complete these preparation steps to ensure you can recover your system if needed and confirm compatibility with the new version:
{.power-number}

1. Create a backup before upgrading, as downgrades are not possible. Therefore, reverting to a previous version requires a backup made prior to the upgrade.

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
    helm upgrade pmm -f values.yaml percona/pmm
    ```

3. After the upgrade, verify that PMM Server is running correctly:

    ```sh
    kubectl get pods | grep pmm-server
    ```

4. Check the logs for any errors:

    ```sh
    kubectl logs deployment/pmm-server
    ```

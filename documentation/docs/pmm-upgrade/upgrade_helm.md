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

1. [Create a backup](../install-pmm/install-pmm-server/baremetal/helm/backup_container_helm.md) before upgrading, as downgrades are not possible. Therefore, reverting to a previous version requires a backup made prior to the upgrade.

2. To reduce downtime, pre-pull the new image on the node where PMM is running:

    ```sh
    # Replace <version> with the latest PMM version
    podman pull percona/pmm-server:3
    ```

## Upgrade steps

Follow these steps to upgrade your PMM Server while preserving your monitoring data and settings—you can restore from your backup if needed.
{.power-number}

1. Stop the current container:

    ```sh
   helm stop pmm-server
    ```

3. Pull the latest image:

    ```sh
   helm pull percona/pmm-server:3
    ```

4. Rename the original container:

    ```sh
   helm rename pmm-server pmm-server-old
    ```

5. Run the new container:

    ```sh
   helm run \
   --detach \
   --restart always \
   --publish 443:8443 \
   --volumes-from pmm-data \
   --name pmm-server \
   percona/pmm-server:3
   ```

6. After upgrading, verify that PMM Server is running correctly and all your data is accessible.
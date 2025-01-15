# Upgrade PMM Server using Podman

## Before you begin

Before starting the upgrade, complete these preparation steps to ensure you can recover your system if needed and confirm compatibility with the new version:
{.power-number}

1. [Create a backup](../install-pmm/install-pmm-server/baremetal/podman/backup_container_podman.md) before upgrading, as downgrades are not possible. Therefore, reverting to a previous version requires an backup made prior to the upgrade.

2. Verify your current PMM version: Check your current PMM version by navigating to **PMM Configuration > Updates** or by running the following command: 

    ```sh
    podman exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

## Upgrade steps

Follow these steps to upgrade your PMM Server while preserving your monitoring data and settings. In case of any issues, you can restore your system using the backup created in the preparation steps.
{.power-number}


1. Stop the current container:

    ```sh
    podman stop pmm-server
    ```

3. Rename the original container:

    ```sh
    podman rename pmm-server pmm-server-old
    ```

4. Run the new container:

    ```sh
    podman run \
    --detach \
    --restart always \
    --publish 443:8443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:3
    ```

5. After the upgrade, verify that PMM Server is running correctly:

    ```sh
    podman ps | grep pmm-server
    ```

6. Check the logs for any errors:

    ```sh
    podman logs pmm-server
    ```
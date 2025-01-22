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


1. Update PMM tag by editing `~/.config/systemd/user/pmm-server.env` file and running the following command to set the latest release version:

    ```sh
    sed -i "s/PMM_IMAGE=.*/PMM_IMAGE=docker.io/percona/pmm-server:3.0.0/g" ~/.config/systemd/user/pmm-server.env
    ```

2. Pre-pull the new image to ensure a faster restart:

    ```sh
    source ~/.config/systemd/user/pmm-server.env
    podman pull ${PMM_IMAGE}:${PMM_TAG}
    ```

3. Restart PMM Server:

    ```sh
    systemctl --user restart pmm-server
    ```

4. After the upgrade, verify that PMM Server is running correctly:

    ```sh
    podman ps | grep pmm-server
    ```
    
5. Check the logs for any errors:

    ```sh
    podman logs pmm-server
    ```
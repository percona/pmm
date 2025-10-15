# Restore PMM Server Podman container

Restore your PMM Server from a backup when you need to recover from an upgrade issue or data corruption, or when migrating to a different system.

??? info "Summary"
    - Stop the PMM Server service
    - Update the PMM Server image reference to the backed-up version
    - Import the backed-up data volume
    - Start the PMM Server service

!!! caution alert alert-warning "Important"
    You must have a [backup](backup_container_podman.md) to restore from.
    Restoration is only necessary if you experience issues with an upgrade or with your monitoring data.

To restore your PMM Server container:
{.power-number}

1. Stop PMM Server:

    ```sh
    systemctl --user stop pmm-server
    ```

2. Run PMM on the previous image, replacing `x.yy.z` with the specific version you were using when you created the backup. Using the same version ensures compatibility with your backup data.

    ```sh
    sed -i "s|PMM_IMAGE=.*|PMM_IMAGE=docker.io/percona/pmm-server:x.yy.z|g" %h/.config/systemd/user/pmm-server.env
    ```

3. Restore the volume:

    ```sh
    podman volume import pmm-server pmm-server-backup.tar
    ```

4. Start PMM Server:

    ```sh
    systemctl --user start pmm-server
    ```

<div hidden>
sleep 30
timeout 60 podman wait --condition=running pmm-server
```
</div>

## Related topics

- [Back up PMM Server Podman container](backup_container_podman.md) 
- [Remove PMM Server Podman container](remove_container_podman.md) 
- [Install PMM Server with Podman](index.md) 
- [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md)

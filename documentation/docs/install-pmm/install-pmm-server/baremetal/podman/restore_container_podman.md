# Restore podman container


??? info "Summary"

    !!! summary alert alert-info ""
        - Stop PMM Server.
        - Run PMM on the previous image.
        - Restore the volume.
        - Start PMM Server.

    ---

!!! caution alert alert-warning "Important"
    You must have a [backup](backup_container_podman.md) to restore from.
    You need to perform restore only if you have issues with upgrade or with the data.

To restore your container:
{.power-number}

1. Stop PMM Server.

    ```sh
    systemctl --user stop pmm-server
    ```

2. Run PMM on the previous image.

    Edit `~/.config/pmm-server/env` file:

    ```sh
    sed -i "s/PMM_TAG=.*/PMM_TAG=2.31.0/g" ~/.config/pmm-server/env
    ```

    !!! caution alert alert-warning "Important"
        X.Y.Z (2.31.0) is the version you used before upgrade and you made Backup with it

3. Restore the volume.

    ```sh
    podman volume import pmm-server pmm-server-backup.tar
    ```

4. Start PMM Server.

    ```sh
    systemctl --user start pmm-server
    ```

    <div hidden>
    sleep 30
    timeout 60 podman wait --condition=running pmm-server
    ```
    </div>



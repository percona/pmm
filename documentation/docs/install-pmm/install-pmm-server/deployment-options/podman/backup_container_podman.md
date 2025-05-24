
# Backup podman container


??? info "Summary"

    !!! summary alert alert-info ""
        - Stop PMM Server.
        - Backup the data.

    ---

!!! caution alert alert-warning "Important"
    Grafana plugins have been moved to the data volume `/srv` since the 2.23.0 version. So if you are upgrading PMM from any version before 2.23.0 and have installed additional plugins then plugins should be installed again after the upgrade.
    To check used grafana plugins: `podman exec -it pmm-server ls /var/lib/grafana/plugins`

To back up your container:
{.power-number}

1. Stop PMM Server.

    ```sh
    systemctl --user stop pmm-server
    ```

2. Backup the data.

    <div hidden>
    ```sh
    podman wait --condition=stopped pmm-server || true
    sleep 30
    ```
    </div>

    ```sh
    podman volume export pmm-server --output pmm-server-backup.tar
    ```

    !!! caution alert alert-warning "Important"
        If you changed the default name to `PMM_VOLUME_NAME` environment variable, use that name after `export` instead of `pmm-server` (which is the default volume name).



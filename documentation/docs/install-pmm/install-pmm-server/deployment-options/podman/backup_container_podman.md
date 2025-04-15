# Back up PMM Server Podman container

Create a backup of your PMM Server data to protect against data loss, prepare for upgrades, or migrate to another system.

??? info "Summary"
    - Stop the PMM Server service
    - Export the data volume to a backup file

## Backing up Grafana plugins

Grafana plugins have been moved to the `/srv` directory since PMM 2.23.0. So if you are upgrading PMM from a version before 2.23.0 and have installed additional plugins, you'll need to reinstall them after the upgrade.
 
To check used Grafana plugins:
```sh
podman exec -t pmm-server ls -l /var/lib/grafana/plugins
```

## Back up procedure

To back up your PMM Server container:
{.power-number}

1. Stop the PMM Server service:

    ```sh
    systemctl --user stop pmm-server
    ```

2. Wait for the container to fully stop:

    <div hidden>
    ```sh
    podman wait --condition=stopped pmm-server || true
    sleep 30
    ```
    </div>

3. Export the data volume to a backup file. If you changed the default name in the `PMM_VOLUME_NAME` environment variable, use that name after export instead of `pmm-server ` (which is the default volume name):

    ```sh
    podman volume export pmm-server --output pmm-server-backup.tar
    ```

4. Verify the backup file was created successfully:
    ```sh
    ls -lh pmm-server-backup.tar
    ```

5. Store the backup in a secure location, preferably outside the current server.

## Backup storage recommendations

- Store backups in a location separate from the PMM Server host
- Implement automated rotation of backups to manage disk space
- Consider encrypting backups containing sensitive monitoring data
- Test restores periodically to verify backup integrity

## Related topics

- [Restore PMM Server Podman container](restore_container_podman.md) 
- [Remove PMM Server Podman container](remove_container_podman.md) 
- [Install PMM Server with Podman](index.md) 
- [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md)
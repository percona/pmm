# Remove PMM Server Podman container
Remove the PMM Server Podman container, images, and data when you no longer need this installation or want to perform a complete reinstallation.

!!! danger alert alert-danger "Data loss warning"
    These steps delete the PMM Server Docker image and the associated PMM metrics data.

To completely remove your container and data:
{.power-number}

1. Stop the PMM Server service:

    ```sh
    systemctl --user stop pmm-server

    # Wait for container to stop completely
    podman wait --condition=stopped pmm-server || true
    sleep 10
    ```

2. If you're using Watchtower for UI upgrades, stop it too:

    ```sh
    systemctl --user stop watchtower
    ```

3. Remove the PMM data volume:

    ```sh
    podman volume rm --force pmm-server
    ```

4. Remove the PMM Server images:

    ```sh
    podman rmi $(podman images | grep "pmm-server" | awk {'print $3'})
    ```

5. Disable the SystemD services:
    ```sh
    systemctl --user disable pmm-server
    systemctl --user disable watchtower
    ```

6. Optionally, remove service files:
    ```sh
    rm -f %h/.config/systemd/user/pmm-server.service
    rm -f %h/.config/systemd/user/pmm-server.env
    rm -f %h/.config/systemd/user/watchtower.service
    rm -f %h/.config/systemd/user/watchtower.env
    ```

7. If you no longer need it, remove the PMM network:
    ```sh
    podman network rm pmm_default
    ```

## Related topics

- [Install PMM Server with Podman](index.md)
- [Back up PMM Server Podman container](backup_container_podman.md) 
- [Restore PMM Server Podman container](restore_container_podman.md) 
- [Install PMM Client](../../../install-pmm-client/index.md) 
- [Uninstall PMM Client](../../../../uninstall-pmm/index.md)


[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Podman]: https://podman.io/getting-started/installation
[Docker]: ../docker/index.md
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker environment variables]: ../docker/env_var.md
# Remove podman container

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Server Docker image and the associated PMM metrics data.

To remove your container:
{.power-number}

1. Stop PMM Server:

    ```sh
    systemctl --user stop pmm-server
    ```

2. Remove volume:

    <div hidden>
    ```sh
    #wait for container to stop
    podman wait --condition=stopped pmm-server || true
    sleep 10
    ```
    </div>

    ```sh
    podman volume rm --force pmm-server
    ```

3. Remove the PMM images:

    ```sh
    podman rmi $(podman images | grep "pmm-server" | awk {'print $3'})
    ```

[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Podman]: https://podman.io/getting-started/installation
[Docker]: ../docker/index.md
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker environment variables]: ../docker/env_var.md
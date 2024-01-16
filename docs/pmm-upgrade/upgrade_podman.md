# Upgrade PMM server using podman

??? info "Summary"

    !!! summary alert alert-info ""
        - Perform a backup.
        - Update PMM tag.
        - Pre-pull image.
        - Run it.

    ---

!!! caution alert alert-warning "Important"
    You cannot downgrade. To go to a previous version, you must create a backup before upgrading.

!!! hint alert alert-success "Tip"
    To see the current release running on your system, use the *PMM Upgrade* panel on the *Home Dashboard*, or run:

    ```sh
    podman exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

(If you are accessing the podman host remotely, replace `localhost` with the IP or server name of the host.)
{.power-number}

1. Perform a [backup](../install-pmm/install-pmm-server/baremetal/podman/backup_container_podman.md).


2. Update PMM tag.

    Edit `~/.config/pmm-server/env` and create/update with a new tag from [latest release](https://per.co.na/pmm/latest):

    ```sh
    sed -i "s/PMM_TAG=.*/PMM_TAG=2.33.0/g" ~/.config/pmm-server/env
    ```

3. Pre-pull image for faster restart.

    <div hidden>
    ```sh
    sed -i "s/PMM_TAG=.*/PMM_TAG=2.33.0-rc/g" ~/.config/pmm-server/env
    sed -i "s|PMM_IMAGE=.*|PMM_IMAGE=docker.io/perconalab/pmm-server|g" ~/.config/pmm-server/env
    ```
    </div>

    ```sh
    source ~/.config/pmm-server/env
    podman pull ${PMM_IMAGE}:${PMM_TAG}
    ```

4. Run PMM.

    ```sh
    systemctl --user restart pmm-server
    ```

<div hidden>
```sh
sleep 30
timeout 60 podman wait --condition=running pmm-server
```
</div>

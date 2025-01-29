# Install PMM Server with Podman on Docker image

This section provides instructions for running PMM Server with Podman based on our [Docker image](https://hub.docker.com/r/percona/pmm-server).

## About Podman

!!! seealso alert alert-info "See also"
    - [Docker](../docker/index.md) 
    - Other [tags](https://hub.docker.com/r/percona/pmm-server/tags) are available.

Podman is an open-source, daemonless container engine for developing, managing, and running Open Container Initiative (OCI) containers and container images on Linux systems. It is widely supported across Linux distributions and hosted on [GitHub](https://github.com/containers/podman).

One of Podman’s advantages is that it allows non-privileged users to run containers, enhancing security by avoiding elevated permissions.

Podman is compatible with Docker; by using an alias (`alias docker=podman`), you can run Docker commands seamlessly with Podman. All instructions in the Docker section apply to Podman as well.

Percona recommends running PMM with Podman as a non-privileged user and as part of the provided SystemD service. SystemD helps ensure that the service is actively running and offers logging and management functions, such as start, stop, and restart.

## Before you start

- Install [Podman](https://podman.io/getting-started/installation).
- Configure [rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md) Podman.
- Install Watchtower to automatically update your containers with the following considerations:

      - Ensure Watchtower is only accessible from within the Docker network or local host to prevent unauthorized access and enhance container security.
      - Configure network settings to expose only the PMM Server container to the external network, keeping Watchtower isolated within the Docker network.
      - Grant Watchtower access to the Docker socket to monitor and manage containers effectively, ensuring proper security measures are in place to protect the Docker socket.
      - Verify that both Watchtower and PMM Server are on the same network, or ensure PMM Server can connect to Watchtower for communication. This network setup is essential for PMM Server to initiate updates through Watchtower.

## Update mechanism

PMM Server updates work differently in Podman compared to Docker due to security policies:

- Docker updates use a simpler flow where PMM Server directly instructs Watchtower to replace the Docker container in one step.
- Podman updates require SystemD integration and follow a multi-step process with environment file changes for better security isolation.

## Install

You can install PMM with either automated UI-based updates or a manual update method, depending on your preferences.

The UI-based method, using Watchtower, enables direct updates from the web interface without requiring command-line access and automates the process. 

On the other hand, the manual method offers a simpler setup with complete control over updates and no need for additional services, but it requires command-line access and manual intervention to track and apply updates.

=== "Installation with UI updates"

    This method enables updates through the PMM web interface using Watchtower and SystemD services. When you initiate an update in the UI, PMM Server updates its image reference, prompting Watchtower to pull the new image. Watchtower then stops the existing container, and SystemD automatically restarts it with the updated image.
    {.power-number}

    1. Create PMM Server service file at `~/.config/systemd/user/pmm-server.service`:

        ```sh
        [Unit]
        Description=pmm-server
        Wants=network-online.target
        After=network-online.target
        After=nss-user-lookup.target nss-lookup.target
        After=time-sync.target
        [Service]
        EnvironmentFile=~/.config/systemd/user/pmm-server.env
        Environment=PMM_VOLUME_NAME=%N
        Restart=on-failure
        RestartSec=20
        ExecStart=/usr/bin/podman run \
            --volume ~/.config/systemd/user/:/home/pmm/update/ \
            --volume=${PMM_VOLUME_NAME}:/srv
            --rm --replace=true --name %N \
            --env-file=~/.config/systemd/user/pmm-server.env \
            --net pmm_default \
            --cap-add=net_admin,net_raw \
            --userns=keep-id:uid=1000,gid=1000 \
            -p 443:8443/tcp --ulimit=host ${PMM_IMAGE}
        ExecStop=/usr/bin/podman stop -t 10 %N
        [Install]
        WantedBy=default.target
        ```

    2. Create the environment file at `~/.config/systemd/user/pmm-server.env`:
   
        ```sh
        PMM_WATCHTOWER_HOST=http://watchtower:8080
        PMM_WATCHTOWER_TOKEN=123
        PMM_IMAGE=docker.io/percona/pmm-server:3
        ```

    3. Create or update the Watchtower service file at `~/.config/systemd/user/watchtower.service`:
   
        ```sh
        [Unit]
        Description=watchtower
        Wants=network-online.target
        After=network-online.target
        After=nss-user-lookup.target nss-lookup.target
        After=time-sync.target
        [Service]
        EnvironmentFile=/home/pmm/watchtower.env
        Restart=on-failure
        RestartSec=20
        ExecStart=/usr/bin/podman run --rm --replace=true --name %N \
            -v ${XDG_RUNTIME_DIR}/podman/podman.sock:/var/run/docker.sock \
            --env-file=~/.config/systemd/user/watchtower.env \
            --net pmm_default \
            --cap-add=net_admin,net_raw \
            ${WATCHTOWER_IMAGE}
        ExecStop=/usr/bin/podman stop -t 10 %N
        [Install]
        WantedBy=default.target
        ```

    4. Create the environment file for Watchtower at `~/.config/systemd/user/watchtower.env`. If running as `root`, modify the permissions as well:
   
        ```sh
        WATCHTOWER_HTTP_API_UPDATE=1
        WATCHTOWER_HTTP_API_TOKEN=123
        WATCHTOWER_NO_RESTART=1
        WATCHTOWER_IMAGE=docker.io/percona/watchtower:latest
        chmod 777 ~/.config/systemd/user/watchtower.env  # Only if running as root
        ```
    
    5. Start services:
   
        ```sh
        systemctl --user enable --now pmm-server
        systemctl --user enable --now watchtower
        ```

    6. Go to `https://localhost:443` to access the PMM user interface in a web browser. If you are accessing the host remotely, replace `localhost` with the IP or server name of the host.

=== "Installation with manual updates"

    The installation with manual updates offers a straightforward setup with direct control over updates, without relying on additional services. In this approach, you manually update the `PMM_IMAGE` in the environment file and restart the PMM Server service. SystemD then automatically manages the container replacement.
    {.power-number}
    
    1. Create PMM Server service file at `~/.config/systemd/user/pmm-server.service`:
   
        ```sh
        [Unit]
        Description=pmm-server
        Wants=network-online.target
        After=network-online.target
        After=nss-user-lookup.target nss-lookup.target
        After=time-sync.target
        [Service]
        EnvironmentFile=~/.config/systemd/user/pmm-server.env
        Environment=PMM_VOLUME_NAME=%N
        Restart=on-failure
        RestartSec=20
        ExecStart=/usr/bin/podman run \
            --volume=${PMM_VOLUME_NAME}:/srv
            --rm --replace=true --name %N \
            --env-file=~/.config/systemd/user/pmm-server.env \
            --net pmm_default \
            --cap-add=net_admin,net_raw \
            --userns=keep-id:uid=1000,gid=1000 \
            -p 443:8443/tcp --ulimit=host ${PMM_IMAGE}
        ExecStop=/usr/bin/podman stop -t 10 %N
        [Install]
        WantedBy=default.target
        ```

    2. Create the environment file at `~/.config/systemd/user/pmm-server.env`:
   
        ```sh
        PMM_IMAGE=docker.io/percona/pmm-server:3
        ```

    3. Start services:
   
        ```sh
        systemctl --user enable --now pmm-server
        ```

    4. Go to `https://localhost:443` to access the PMM user interface in a web browser. If you are accessing the host remotely, replace `localhost` with the IP or server name of the host.

    For information on manually upgrading, see [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md).

<div hidden>
```sh
#first pull can take time
sleep 80
timeout 60 podman wait --condition=running pmm-server
```
</div>

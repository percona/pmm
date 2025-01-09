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
        Restart=on-failure
        RestartSec=20
        ExecStart=/usr/bin/podman run \
            --volume ~/.config/systemd/user/:/home/pmm/update/ \
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
        Restart=on-failure
        RestartSec=20
        Environment=WATCHTOWER_HTTP_API_UPDATE=1
        Environment=WATCHTOWER_HTTP_API_TOKEN=123
        Environment=WATCHTOWER_NO_RESTART=1
        Environment=WATCHTOWER_DEBUG=1
        ExecStart=/usr/bin/podman run --rm --replace=true --name %N \
            -v ${XDG_RUNTIME_DIR}/podman/podman.sock:/var/run/docker.sock \
            -e WATCHTOWER_HTTP_API_UPDATE=${WATCHTOWER_HTTP_API_UPDATE} \
            -e WATCHTOWER_HTTP_API_TOKEN=${WATCHTOWER_HTTP_API_TOKEN} \
            -e WATCHTOWER_NO_RESTART=${WATCHTOWER_NO_RESTART} \
            -e WATCHTOWER_DEBUG=${WATCHTOWER_DEBUG} \
            --net pmm_default \
            --cap-add=net_admin,net_raw \
            docker.io/perconalab/watchtower:latest
        ExecStop=/usr/bin/podman stop -t 10 %N
        [Install]
        WantedBy=default.target
        ```

    4. Start services:
   
        ```sh
        systemctl --user enable --now pmm-server
        systemctl --user enable --now watchtower
        ```

    5. Go to `https://localhost:8443` to access the PMM user interface in a web browser. If you are accessing the host remotely, replace `localhost` with the IP or server name of the host.

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
        Restart=on-failure
        RestartSec=20
        ExecStart=/usr/bin/podman run \
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

    4. Go to `https://localhost:8443` to access the PMM user interface in a web browser. If you are accessing the host remotely, replace `localhost` with the IP or server name of the host.

    For information on manually upgrading, see [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md).


## Run as non-privileged user to start PMM

??? info "Summary"

    !!! summary alert alert-info ""
        - Install.
        - Configure.
        - Enable and Start.
        - Open the PMM UI in a browser.

    ---
To run Podman as a non-privileged user:
{.power-number}

1. Install:

    Create `~/.config/systemd/user/pmm-server.service` file:

    ```sh
    mkdir -p ~/.config/systemd/user/
    cat << "EOF" > ~/.config/systemd/user/pmm-server.service
    [Unit]
    Description=pmm-server
    Wants=network-online.target
    After=network-online.target
    After=nss-user-lookup.target nss-lookup.target
    After=time-sync.target

    [Service]
    Type=simple

    # set environment for this unit
    Environment=PMM_PUBLIC_PORT=8443
    Environment=PMM_VOLUME_NAME=%N
    Environment=PMM_TAG=2.33.0
    Environment=PMM_IMAGE=docker.io/percona/pmm-server
    Environment=PMM_ENV_FILE=%h/.config/pmm-server/pmm-server.env

    # optional env file that could override previous env settings for this unit
    EnvironmentFile=-%h/.config/pmm-server/env

    ExecStart=/usr/bin/podman run --rm --replace=true --name=%N -p ${PMM_PUBLIC_PORT}:443/tcp --ulimit=host --volume=${PMM_VOLUME_NAME}:/srv --env-file=${PMM_ENV_FILE} --health-cmd=none --health-interval=disable ${PMM_IMAGE}:${PMM_TAG}
    ExecStop=/usr/bin/podman stop -t 10 %N
    Restart=on-failure
    RestartSec=20

    [Install]
    Alias=%N
    WantedBy=default.target

    EOF
    ```

    Create `~/.config/pmm-server/pmm-server.env` file:

    ```sh
    mkdir -p ~/.config/pmm-server/
    cat << "EOF" > ~/.config/pmm-server/pmm-server.env
    # env file passed to the container
    # full list of environment variables:
    # https://www.percona.com/doc/percona-monitoring-and-management/2.x/setting-up/server/docker.html#environment-variables

    # keep updates disabled
    # do image replacement instead (update the tag and restart the service)
    DISABLE_UPDATES=1
    EOF
    ```

2. Configure:

    There are 2 configuration files:
    1.  `~/.config/pmm-server/pmm-server.env` defines environment variables for PMM Server (PMM parameters like RBAC feature and etc)
    2.  `~/.config/pmm-server/env` defines environment variables for SystemD service (image tags, repo and etc)

    SystemD service passes the environment parameters from the `pmm-server.env `file (in `~/.config/pmm-server/pmm-server.env`) to PMM. For more information about container environment variables, check [Docker Environment].

    SystemD service uses some environment variables that could be customized if needed:

    ```text
    Environment=PMM_PUBLIC_PORT=8443
    Environment=PMM_VOLUME_NAME=%N
    Environment=PMM_TAG=2.33.0
    Environment=PMM_IMAGE=docker.io/percona/pmm-server
    ```

    You can override the environment variables by defining them in the file  `~/.config/pmm-server/env`. For example, to override the path to a custom registry `~/.config/pmm-server/env`:

    ```sh
    mkdir -p ~/.config/pmm-server/
    cat << "EOF" > ~/.config/pmm-server/env
    PMM_TAG=2.31.0
    PMM_IMAGE=docker.io/percona/pmm-server
    PMM_PUBLIC_PORT=8443
    EOF
    ```

    !!! caution alert alert-warning "Important"
        Ensure that you modify PMM_TAG in `~/.config/pmm-server/env` and update it regularly as Percona cannot update it. It needs to be done by you.

3. Enable and start:

    ```sh
    systemctl --user enable --now pmm-server
    ```

4. Activate the podman socket using the [Podman socket activation instructions](https://github.com/containers/podman/blob/main/docs/tutorials/socket_activation.md).

5. Pass the following command to Docker Socket to start [Watchtower](https://containrrr.dev/watchtower/). Make sure to modify the command to use your Podman socket path:

    ```sh
    docker  run -v $XDG_RUNTIME_DIR/podman/podman.sock:/var/run/docker.sock -e WATCHTOWER_HTTP_API_UPDATE=1 -e WATCHTOWER_HTTP_API_TOKEN=123 --hostname=watchtower --network=pmm_default docker.io/perconalab/watchtower
    ```

6. Visit `https://localhost:8443` to see the PMM user interface in a web browser. (If you are accessing host remotely, replace `localhost` with the IP or server name of the host.)

<div hidden>
```sh
#first pull can take time
sleep 80
timeout 60 podman wait --condition=running pmm-server
```
</div>

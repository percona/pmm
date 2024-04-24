# Install PMM server with Podman on Docker image

This section provides instructions for running PMM Server with Podman based on our [Docker image](https://hub.docker.com/r/percona/pmm-server).

!!! note alert alert-primary ""
    The tags used here are for the current release (PMM 2.33.0). Other [tags](https://hub.docker.com/r/percona/pmm-server/tags) are available.

!!! seealso alert alert-info "See also"
    [Docker](../docker/index.md)

Podman is an open-source project available on most Linux platforms and resides on [GitHub](https://github.com/containers/podman). Podman is a daemonless container engine for developing, managing, and running Open Container Initiative (OCI) containers and container images on your Linux System. 

Non-privileged users could run containers under the control of Podman.

It could be just aliased (`alias docker=podman`) with docker and work with the same way. All instructions from [Docker](../docker/index.md) section also apply here.

Percona recommends running PMM as a non-privileged user and running it as part of the SystemD service provided. SystemD service ensures that the service is running and maintains logs and other management features (start, stop, etc.).

## Before you start

- Install [Podman](https://podman.io/getting-started/installation).
- Configure [rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md) Podman.
- Install Watchtower to automatically update your containers with the following considerations:

      - Ensure Watchtower is only accessible from within the Docker network or local host to prevent unauthorized access and enhance container security.
      - Configure network settings to expose only the PMM Server container to the external network, keeping Watchtower isolated within the Docker network.
      - Grant Watchtower access to the Docker socket to monitor and manage containers effectively, ensuring proper security measures are in place to protect the Docker socket.
      - Verify that both Watchtower and PMM Server are on the same network, or ensure PMM Server can connect to Watchtower for communication. This network setup is essential for PMM Server to initiate updates through Watchtower.

## Run as non-privileged user to start PMM

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.

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

    # Enable DBaaS feature
    #ENABLE_DBAAS=1
    EOF
    ```

2. Configure:

    There are 2 configuration files:
    1.  `~/.config/pmm-server/pmm-server.env` defines environment variables for PMM Server (PMM parameters like DBaaS feature and etc)
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
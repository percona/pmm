# Podman

How to run PMM Server with Podman on our [Docker image]

!!! note alert alert-primary ""
    The tags used here are for the current release (PMM 2.33.0). Other [tags] are available.

!!! seealso alert alert-info "See also"
    [Docker]

Podman is an open-source project available on most Linux platforms and resides on [GitHub](https://github.com/containers/podman). Podman is a daemonless container engine for developing, managing, and running Open Container Initiative (OCI) containers and container images on your Linux System. 

Non-privileged users could run containers under the control of Podman.

It could be just aliased (`alias docker=podman`) with docker and work with the same way. All instructions from [Docker] section also apply here.

Percona recommends running PMM as a non-privileged user and running it as part of the SystemD service provided. SystemD service ensures that the service is running and maintains logs and other management features (start, stop, etc.).

## Before you start

- Install [Podman](https://podman.io/getting-started/installation).
- Configure [rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)  Podman.

## Run as non-privileged user to start PMM

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.

!!! summary alert alert-info "Summary"
    - Install.
    - Configure.
    - Enable and Start.
    - Open the PMM UI in a browser.

---

1. Install.

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

2. Configure.

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

3. Enable and Start.

    ```sh
    systemctl --user enable --now pmm-server
    ```

4. Visit `https://localhost:8443` to see the PMM user interface in a web browser. (If you are accessing host remotely, replace `localhost` with the IP or server name of the host.)

<div hidden>
```sh
#first pull can take time
sleep 80
timeout 60 podman wait --condition=running pmm-server
```
</div>

## Backup

!!! summary alert alert-info "Summary"
    - Stop PMM server.
    - Backup the data.

---

!!! caution alert alert-warning "Important"
    Grafana plugins have been moved to the data volume `/srv` since the 2.23.0 version. So if you are upgrading PMM from any version before 2.23.0 and have installed additional plugins then plugins should be installed again after the upgrade.
    To check used grafana plugins: `podman exec -it pmm-server ls /var/lib/grafana/plugins`

1. Stop PMM server.

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

## Upgrade

!!! summary alert alert-info "Summary"
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

1. Perform a [backup](#backup).


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

## Restore

!!! summary alert alert-info "Summary"
    - Stop PMM server.
    - Run PMM on the previous image.
    - Restore the volume.
    - Start PMM Server.

---

!!! caution alert alert-warning "Important"
    You must have a [backup](#backup) to restore from.
    You need to perform restore only if you have issues with upgrade or with the data.

1. Stop PMM server.

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

## Remove

!!! summary alert alert-info "Summary"
    - Stop PMM server.
    - Remove (delete) volume.
    - Remove (delete) images.

---

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Server Docker image and the associated PMM metrics data.

1. Stop PMM server.

    ```sh
    systemctl --user stop pmm-server
    ```

2. Remove volume.

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

3. Remove the PMM images.

    ```sh
    podman rmi $(podman images | grep "pmm-server" | awk {'print $3'})
    ```

[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Podman]: https://podman.io/getting-started/installation
[Docker]: docker.md
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker Environment]: docker.md#environment-variables
[trusted certificate]: ../../how-to/secure.md#ssl-encryption
[Set up repos]: ../client/index.md#package-manager

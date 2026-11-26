# Install PMM Server with Podman 

Run PMM Server with Podman based on our [Docker image](https://hub.docker.com/r/percona/pmm-server) when you need enhanced security, rootless container execution, or are working in environments where Docker daemon is not preferred. Podman provides improved security isolation while maintaining compatibility with Docker commands and workflows.

## About Podman

Podman is an open-source, daemonless container engine for developing, managing, and running Open Container Initiative (OCI) containers and container images on Linux systems. It is widely supported across Linux distributions and hosted on [GitHub](https://github.com/containers/podman).

One of Podman's main benefits is that non-privileged users can run containers without elevated permissions. Podman is largely compatible with Docker commands. If needed, you can set an alias (`alias docker=podman`) and use familiar Docker workflows. 

Most Docker-based PMM steps work with Podman, but follow the Podman-specific update and systemd instructions in this topic.

Choose Podman deployment when:

- security is a priority and you need rootless container execution
- your organization has security policies restricting the use of Docker daemon
- you're running in environments where fine-grained permission control is required
- you need systemd integration for better service management


!!! tip "Recommended setup for best performance"
    Percona recommends running PMM with Podman as a non-privileged user and as part of the provided systemd service. Systemd helps ensure that the service is actively running and offers logging and management functions, such as start, stop, and restart.

## Before you start

Before installing PMM Server with Podman, ensure you have:
{.power-number}

1. Install [Podman](https://podman.io/getting-started/installation).
2. Configure [rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md) Podman.
3. Create the Podman volume for PMM:
  ```sh
  podman volume create pmm-data
  ```
4. Create the Podman network for PMM:
  ```sh
  podman network create pmm_default
  ```
5. Set up required system configurations:
    ```sh
    # Allow non-root users to bind to privileged ports (required for port 443)        
    # Make the setting persistent
    echo "net.ipv4.ip_unprivileged_port_start=443" | sudo tee /etc/sysctl.d/99-pmm.conf
    sudo sysctl -p /etc/sysctl.d/99-pmm.conf
    ```

## Log driver compatibility
When you run PMM Server as a systemd service with Podman, the default log driver is `journald`. This makes container logs persistent and queryable via `journalctl`.

Avoid `--log-driver passthrough`. Podman's passthrough driver passes the host's stdio file descriptors directly into the container, which makes `/dev/stdin`, `/dev/stdout`, and `/dev/stderr` unavailable inside the container. 

PMM Server references `/dev/stderr` in its nginx configuration, so the startup check fails with:

```
[emerg] open() "/dev/stderr" failed (6: No such device or address)
nginx: configuration file /etc/nginx/nginx.conf test failed
```
    
Use one of these log drivers instead:

- `journald`: persistent logs, queryable with `journalctl -u pmm-server` (recommended)
- `none`: disables logging

## Install PMM Server as a systemd service

To upgrade PMM Server, you manually update the `PMM_IMAGE` in the environment file and restart the PMM Server service. Systemd then automatically manages the container replacement.
{.power-number}

1. Create directories for configuration files if they don't exist:

    ```sh
    mkdir -p ~/.config/systemd/user/
    ```

2. Create PMM Server service file at `~/.config/systemd/user/pmm-server.service`:

    ```sh
    [Unit]
    Description=pmm-server
    Wants=network-online.target
    After=network-online.target
    After=nss-user-lookup.target nss-lookup.target
    After=time-sync.target
    [Service]
    EnvironmentFile=%h/.config/systemd/user/pmm-server.env
    Environment=PMM_VOLUME_NAME=pmm-data
    TimeoutStartSec=480
    Restart=on-failure
    RestartSec=20
    ExecStart=/usr/bin/podman run \
        --volume=${PMM_VOLUME_NAME}:/srv \
        --rm --replace=true --name %N \
        --env-file=%h/.config/systemd/user/pmm-server.env \
        --net pmm_default \
        --cap-add=net_admin,net_raw \
        --userns=keep-id:uid=1000,gid=1000 \
        -p 443:8443/tcp --ulimit=host ${PMM_IMAGE}
    ExecStop=/usr/bin/podman stop -t 10 %N
    [Install]
    WantedBy=default.target
    ```

3. Create the environment file at `~/.config/systemd/user/pmm-server.env`:

    ```sh
    PMM_IMAGE=docker.io/percona/pmm-server:3
    ```

4. Enable and start the PMM Server service:

    ```sh
    systemctl --user enable --now pmm-server
    ```

5. Go to `https://localhost:443` to access the PMM user interface in a web browser. If you are accessing the host remotely, replace `localhost` with the IP or server name of the host.

For information on manually upgrading, see [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md).


## Related topics

- [Docker installation alternative](../docker/index.md) 
- [Available image tags](https://hub.docker.com/r/percona/pmm-server/tags)
- [Upgrade PMM Server using Podman](../../../../pmm-upgrade/upgrade_podman.md) 
- [Back up PMM Server Podman container](backup_container_podman.md) 
- [Restore PMM Server Podman container](restore_container_podman.md)
- [Remove PMM Server Podman container](remove_container_podman.md) 
- [Install PMM Client](../../../install-pmm-client/index.md) 

<div hidden>
```sh
# first pull can take time
timeout 60 podman wait --condition=running pmm-server
```
</div>

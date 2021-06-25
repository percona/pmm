# Docker

How to run PMM Server with Docker based on our [Docker image].

!!! note alert alert-primary ""
    The tags used here are for the current release. Other [tags] are available.

!!! seealso alert alert-info "See also"
    [Easy-install script]

## Before you start

- Install [Docker] 1.12.6 or higher.
- (Optional) Install [Docker compose].

## Run

!!! summary alert alert-info "Summary"
    - Pull the Docker image.
    - Copy it to create a persistent data container.
    - Run the image.
    - Open the PMM UI in a browser.

---

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Create a persistent data container.

    ```sh
    docker create --volume /srv \
    --name pmm-data \
    percona/pmm-server:2 /bin/true
    ```

    !!! caution alert alert-warning "Important"
        PMM Server expects the data volume to be `/srv`. Using any other value will result in **data loss** when upgrading.

        To check server and data container mount points:

        ```sh
        docker inspect pmm-data | grep Destination && \
        docker inspect pmm-server | grep Destination
        ```

3. Run the image.

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:2
    ```

4. In a web browser, visit `https://localhost:443` (or `http://localhost:80` if enabled) to see the PMM user interface. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

## Backup

!!! summary alert alert-info "Summary"
    - Stop and rename the `pmm-server` container.
    - Take a local copy of the `pmm-data` container's `/srv` directory.

---

1. Stop the container.

    ```sh
    docker stop pmm-server
    ```

1. Move the image.

    ```sh
    docker rename pmm-server pmm-server-backup
    ```

1. Create a subdirectory (e.g., `pmm-data-backup`) and move to it.

    ```sh
    mkdir pmm-data-backup && cd pmm-data-backup
    ```

1. Backup the data.

    ```sh
    docker cp pmm-data:/srv .
    ```

## Upgrade

!!! summary alert alert-info "Summary"
    - Stop the running container.
    - Backup (rename) the container and copy data.
    - Pull the latest Docker image.
    - Run it.

---

!!! hint alert alert-success "Tip"
    To see what release you are running, use the *PMM Upgrade* panel on the *Home Dashboard*, or run:

    ```sh
    docker exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

    (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

1. Perform a [backup](#backup).

2. Pull the latest image.

    ```sh
    docker pull percona/pmm-server:2
    ```

3. Run it.

    ```sh
    docker run \
    --detach \
    --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:2
    ```

## Restore

!!! summary alert alert-info "Summary"
    - Stop and remove the container.
    - Restore (rename) the backup container.
    - Restore saved data to the data container.
    - Restore permissions to the data.

---

!!! caution alert alert-warning "Important"
    You must have a [backup](#backup) to restore from.

1. Stop the container.

    ```sh
    docker stop pmm-server
    ```

1. Remove it.

    ```sh
    docker rm pmm-server
    ```

1. Revert to the saved image.

    ```sh
    docker rename pmm-server-backup pmm-server
    ```

1. Change directory to the backup directory (e.g. `pmm-data-backup`).

1. Copy the data.

    ```sh
    docker cp srv pmm-data:/
    ```

1. Restore permissions.

    ```sh
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:root /srv && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/alertmanager && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:pmm /srv/clickhouse && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R grafana:grafana /srv/grafana && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/logs && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/postgres && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/prometheus && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/victoriametrics && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/logs/postgresql.log
    ```

1. Start the image.

    ```sh
    docker start pmm-server
    ```

## Remove

!!! summary alert alert-info "Summary"
    - Stop the container.
    - Remove (delete) both the server and data containers.
    - Remove (delete) both images.

---

!!! caution alert alert-warning "Caution"
    These steps delete the PMM Server Docker image and any accumulated PMM metrics data.

1. Stop pmm-server container.

    ```sh
    docker stop pmm-server
    ```

1. Remove containers.

    ```sh
    docker rm pmm-server pmm-data
    ```

1. Remove the image.

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk {'print $3'})
    ```

## Docker compose {: #docker-compose }

!!! summary alert alert-info "Summary"
    - Copy and paste the `docker-compose.yml` file.
    - Run `docker-compose up`.

---

!!! note alert alert-primary ""
    With this approach, data is stored in a volume, not in a `pmm-data` container.

1. Copy and paste this text into a file called `docker-compose.yml`.

    ```yaml
    version: '2'
    services:
      pmm-server:
        image: percona/pmm-server:2
        hostname: pmm-server
        container_name: pmm-server
        restart: always
        logging:
          driver: json-file
          options:
            max-size: "10m"
            max-file: "5"
        ports:
          - "443:443"
        volumes:
          - data:/srv
    volumes:
      data:
    ```

1. Run:

    ```sh
    docker-compose up
    ```

1. In a web browser, visit `https://localhost:443` to see the PMM user interface. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

## Environment variables

Use the following Docker container environment variables (with `-e var=value`) to set PMM Server parameters.

| Variable                   | Description                                                             |
| -------------------------- | ----------------------------------------------------------------------- |
| `DISABLE_UPDATES`          | Disable automatic updates                                               |
| `DISABLE_TELEMETRY`        | Disable built-in telemetry and disable STT if telemetry is disabled     |
| `METRICS_RESOLUTION`       | High metrics resolution in seconds                                      |
| `METRICS_RESOLUTION_HR`    | High metrics resolution (same as above)                                 |
| `METRICS_RESOLUTION_MR`    | Medium metrics resolution in seconds                                    |
| `METRICS_RESOLUTION_LR`    | Low metrics resolution in seconds                                       |
| `DATA_RETENTION`           | How many days to keep time-series data in ClickHouse                    |
| `ENABLE_VM_CACHE`          | Enable cache in VM                                                      |
| `ENABLE_ALERTING`          | Enable integrated alerting                                              |
| `ENABLE_AZUREDISCOVER`     | Enable support for discovery of Azure databases                         |
| `ENABLE_BACKUP_MANAGEMENT` | Enable integrated backup tools                                          |
| `PERCONA_TEST_SAAS_HOST`   | SaaS server hostname                                                    |
| `PERCONA_TEST_DBAAS`       | Enable testing DBaaS features. (Will be deprecated in future versions.) |
| `ENABLE_DBAAS`             | Enable DBaaS features                                                   |
| `PMM_DEBUG`                | Enables a more verbose log level                                        |
| `PMM_TRACE`                | Enables a more verbose log level including trace-back information       |

### Ignored variables

These variables will be ignored by `pmm-managed` when starting the server. If any other variable is found, it will be considered invalid and the server won't start.

| Variable                                                        | Description                                            |
| --------------------------------------------------------------- | ------------------------------------------------------ |
| `_`, `HOME`, `HOSTNAME`, `LANG`, `PATH`, `PWD`, `SHLVL`, `TERM` | Default environment variables                          |
| `GF_*`                                                          | Grafana's environment variables                        |
| `SUPERVISOR_`                                                   | `supervisord` environment variables                    |
| `PERCONA_TEST_`                                                 | Unknown variable but won't prevent the server starting |
| `PERCONA_TEST_DBAAS`                                            | Deprecated. Use `ENABLE_DBAAS`                         |

## Tips

- Disable manual updates via the Home Dashboard *PMM Upgrade* panel by adding `-e DISABLE_UPDATES=true` to the `docker run` command.

- Eliminate browser certificate warnings by configuring a [trusted certificate].

- Optionally enable an (insecure) HTTP connection by adding `--publish 80:80` to the `docker run` command. However note that PMM Client *requires* TLS to communicate with the server so will only work on the secure port.

### Isolated hosts

If the host where you will run PMM Server has no internet connection, you can download the Docker image on a separate (internet-connected) host and securely copy it.

1. On an internet-connected host, download the Docker image and its checksum file.

    ```sh
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/docker/pmm-server-{{release}}.docker
    wget https://downloads.percona.com/downloads/pmm2/{{release}}/docker/pmm-server-{{release}}.sha256sum
    ```

2. Copy both files to where you will run PMM Server.

3. Open a terminal on the PMM Server host.

4. (Optional) Check the Docker image file integrity.

    ```sh
    shasum -ca 256 pmm-server-{{release}}.sha256sum
    ```

5. Load the image.

    ```sh
    docker load -i pmm-server-{{release}}.docker
    ```

6. Create the `pmm-data` persistent data container.

    ```sh
    docker create --volume /srv \
    --name pmm-data percona/pmm-server:{{release}} /bin/true
    ```

7. Run the container.

    ```sh
    docker run \
    --detach \
    --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:{{release}}
    ```

[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Docker]: https://docs.docker.com/get-docker/
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker compose]: https://docs.docker.com/compose/
[PMMC_COMPOSE]: ../client/index.md#docker-compose
[trusted certificate]: ../../how-to/secure.md#ssl-encryption
[Easy-install script]: easy-install.md
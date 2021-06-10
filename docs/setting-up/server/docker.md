# Docker

We maintain a [Docker image for PMM Server][DOCKERHUB]. This section shows how to run PMM Server as a Docker container, directly and with [Docker compose](#docker-compose). (The tags used here are for the latest version of PMM 2 ({{release}}). [Other tags are available][TAGS].)

## System requirements

**Software**

- [Docker](https://docs.docker.com/get-docker/) 1.12.6 or higher.
- (Optional) [Docker compose](https://docs.docker.com/compose/install/)

## Running PMM Server with Docker {: #docker }

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Create a persistent data container.

    ```sh
    docker create --volume /srv \
    --name pmm-data percona/pmm-server:2 /bin/true
    ```

    PMM Server expects the data volume (specified with `--volume`) to be `/srv`.  **Using any other value will result in data loss when upgrading.**

3. Run the image to start PMM Server.

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    --volumes-from pmm-data --name pmm-server \
    percona/pmm-server:2
    ```

    !!! note alert alert-primary "Note"
        Optionally you can enable http (insecure) by including `--publish 80:80` in the above docker run command however note that PMM Client *requires* TLS to communication with the server so will only work on the secure port.

    You can disable manual updates via the Home Dashboard *PMM Upgrade* panel by adding `-e DISABLE_UPDATES=true` to the `docker run` command.


4. In a web browser, visit *https://server-hostname*:443 (or *http://server-hostname*:80 if optionally enabled) to see the PMM user interface.

    !!! tip alert alert-success "Tip"
        Eliminate browser certificate warnings by configuring a [trusted certificate](https://www.percona.com/doc/percona-monitoring-and-management/2.x/how-to/secure.html#ssl-encryption)

### Docker environment variables

It is possible to change some server setting by using environment variables when starting the Docker container.
Use `-e var=value` in your pmm-server run command.

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

#### Ignored variables

These variables will be ignored by `pmm-managed` when starting the server. If any other variable is found, it will be considered invalid and the server won't start.

| Variable                                                        | Description                                            |
| --------------------------------------------------------------- | ------------------------------------------------------ |
| `_`, `HOME`, `HOSTNAME`, `LANG`, `PATH`, `PWD`, `SHLVL`, `TERM` | Default environment variables                          |
| `GF_*`                                                          | Grafana's environment variables                        |
| `SUPERVISOR_`                                                   | Supervisord environment variables                      |
| `PERCONA_TEST_`                                                 | Unknown variable but won't prevent the server starting |
| `PERCONA_TEST_DBAAS`                                            | Deprecated. Use `ENABLE_DBAAS`                         |

## Backup and upgrade

You can test a new release of the PMM Server Docker image by making backups of your current `pmm-server` and `pmm-data` containers which you can restore if you need to.

1. Find out which release you have now.

    ```sh
    docker exec -it pmm-server curl -u admin:admin https://localhost/v1/version
    ```

    !!! tip alert alert-success "Tip"
        Use `jq` to extract the quoted string value.
        ```sh
        apt install jq # Example for Debian, Ubuntu
        docker exec -it pmm-server curl -u admin:admin https://localhost/v1/version | jq .version
        ```

2. Check the container mount points are the same (`/srv`).

    ```sh
    docker inspect pmm-data | grep Destination
    docker inspect pmm-server | grep Destination
    ```

    With `jq`:

    ```sh
    docker inspect pmm-data | jq '.[].Mounts[].Destination'
    docker inspect pmm-server | jq '.[].Mounts[].Destination'
    ```

3. Stop the container and create backups.

    ```sh
    docker stop pmm-server
    docker rename pmm-server pmm-server-backup
    mkdir pmm-data-backup && cd $_
    docker cp pmm-data:/srv .
    ```

4. Pull the latest image and run the container.

    ```sh
    docker pull percona/pmm-server:2
    docker run \
    --detach \
    --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:2
    ```

5. (Optional) Repeat step 1 to confirm the version, or check the *PMM Upgrade* panel on the *Home Dashboard*.

## Restore

1. Stop and remove the running version.

    ```sh
    docker stop pmm-server
    docker rm pmm-server
    ```

2. Restore backups.

    ```sh
    docker rename pmm-server-backup pmm-server
    # cd to wherever you saved the backup
    docker cp srv pmm-data:/
    ```

3. Restore permissions.

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

4. Start the image.

    ```sh
    docker start pmm-server
    ```

## Running PMM Server with Docker compose {: #docker-compose }

!!! important alert alert-success "Important"
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

2. Run:

    ```sh
    docker-compose up
    ```

3. Access PMM Server on <https://X.X.X.X:443> where `X.X.X.X` is the IP address of the PMM Server host.

!!! seealso alert alert-info "See also"
    [Run PMM Client with Docker compose][PMMC_COMPOSE]

## Removing PMM Server

1. Stop pmm-server container.

    ```sh
    docker stop pmm-server
    ```

2. Remove containers.

    ```sh
    docker rm pmm-server pmm-data
    ```

3. Remove the image.

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk {'print $3'})
    ```


## Hosts with no internet connectivity

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



[TAGS]: https://hub.docker.com/r/percona/pmm-server/tags
[DOCKERHUB]: https://hub.docker.com/r/percona/pmm-server
[DOCKER_COMPOSE]: https://docs.docker.com/compose/
[PMMC_COMPOSE]: ../client/index.md#docker-compose

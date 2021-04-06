# Docker

We maintain a [Docker image for PMM Server][DOCKERHUB]. This section shows how to run PMM Server as a Docker container, directly and with Docker compose. (The tags used here are for the latest version of PMM 2 ({{release}}). [Other tags are available][TAGS].)

## System requirements

**Software**

- [Docker](https://docs.docker.com/get-docker/) 1.12.6 or higher.
- (Optional) [`docker-compose`](https://docs.docker.com/compose/install/)

## Running PMM Server with Docker {: #setting-up-server-docker }

1. Pull the image.

    ```sh
    sudo docker pull percona/pmm-server:2
    ```

1. Create a persistent data container.

    ```sh
    sudo docker create --volume /srv \
    --name pmm-data percona/pmm-server:2 /bin/true
    ```

    PMM Server expects the data volume (specified with `--volume`) to be `/srv`.  **Using any other value will result in data loss when upgrading.**

1. Run the image to start PMM Server.

    ```sh
    sudo docker run --detach --restart always \
    --publish 80:80 --publish 443:443 \
    --volumes-from pmm-data --name pmm-server \
    percona/pmm-server:2
    ```

    You can disable manual updates via the Home Dashboard *PMM Upgrade* panel by adding `-e DISABLE_UPDATES=true` to the `docker run` command.

1. In a web browser, visit *server hostname*:80 or *server hostname*:443 to see the PMM user interface.

## Backup and upgrade

You can test a new release of the PMM Server Docker image by making backups of your current `pmm-server` and `pmm-data` containers which you can restore if you need to.

1. Find out which release you have now.

    ```sh
    sudo docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version
    ```

	> **Tip:** Use `jq` to extract the quoted string value.
	> ```sh
	> sudo apt install jq # Example for Debian, Ubuntu
	> sudo docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version | jq .version
	> ```

2. Check the container mount points are the same (`/srv`).

    ```sh
    sudo docker inspect pmm-data | grep Destination
    sudo docker inspect pmm-server | grep Destination
    ```

    With `jq`:

    ```sh
    sudo docker inspect pmm-data | jq '.[].Mounts[].Destination'
    sudo docker inspect pmm-server | jq '.[].Mounts[].Destination'
    ```

3. Stop the container and create backups.

    ```sh
    sudo docker stop pmm-server
    sudo docker rename pmm-server pmm-server-backup
    mkdir pmm-data-backup && cd $_
    sudo docker cp pmm-data:/srv .
    ```

4. Pull and run the latest image.

    ```sh
    sudo docker pull percona/pmm-server:2
    sudo docker run \
    --detach \
    --restart always \
    --publish 80:80 --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:2
    ```

5. (Optional) Repeat step 1 to confirm the version, or check the *PMM Upgrade* panel on the *Home Dashboard*.

## Restore

1. Stop and remove the running version.

    ```sh
    sudo docker stop pmm-server
    sudo docker rm pmm-server
    ```

2. Restore backups.

    ```sh
    sudo docker rename pmm-server-backup pmm-server
    # cd to wherever you saved the backup
    sudo docker cp srv pmm-data:/
    ```

3. Restore permissions.

    ```sh
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:root /srv && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/alertmanager && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:pmm /srv/clickhouse && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R grafana:grafana /srv/grafana && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/logs && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/postgres && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/prometheus && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/victoriametrics && \
    sudo docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/logs/postgresql.log
    ```

4. Start (don’t run) the image.

    ```sh
    sudo docker start pmm-server
    ```

## Running PMM Server with Docker compose {: #docker-compose }

<!-- thanks: https://gist.github.com/paskal -->

> With this approach, data is stored in a volume, not in a `pmm-data` container.

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
    sudo docker-compose up
    ```

3. Access PMM Server on <https://X.X.X.X:443> where `X.X.X.X` is the IP address of the PMM Server host.

> **See also** [Run PMM Client with Docker compose][PMMC_COMPOSE]

[TAGS]: https://hub.docker.com/r/percona/pmm-server/tags
[DOCKERHUB]: https://hub.docker.com/r/percona/pmm-server
[DOCKER_COMPOSE]: https://docs.docker.com/compose/
[PMMC_COMPOSE]: ../client/index.md#docker-compose

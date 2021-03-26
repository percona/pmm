# Docker

We maintain a [Docker image for PMM Server][DOCKERHUB]. This section shows how to run PMM Server as a Docker container. (The tags used here are for the latest version of PMM 2 ({{release}}). [Other tags are available][TAGS].)

## System requirements

**Software**

- [Docker](https://docs.docker.com/get-docker/) 1.12.6 or higher.

## Running PMM Server as a Docker container

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

1. Create a persistent data container.

    ```sh
    docker create --volume /srv \
    --name pmm-data percona/pmm-server:2 /bin/true
    ```

    PMM Server expects the data volume (specified with `--volume`) to be `/srv`.  **Using any other value will result in data loss when upgrading.**

1. Run the image to start PMM Server.

    ```sh
    docker run --detach --restart always \
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
    docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version
    ```

	> **Tip:** Use `jq` to extract the quoted string value.
	> ```sh
	> sudo apt install jq # Example for Debian, Ubuntu
	> docker exec -it pmm-server curl -u admin:admin http://localhost/v1/version | jq .version
	> ```

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

4. Pull and run the latest image.

    ```sh
    docker pull percona/pmm-server:2
    docker run \
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

4. Start (don’t run) the image.

    ```sh
    docker start pmm-server
    ```


[TAGS]: https://hub.docker.com/r/percona/pmm-server/tags
[DOCKERHUB]: https://hub.docker.com/r/percona/pmm-server

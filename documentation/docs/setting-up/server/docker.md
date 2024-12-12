# Docker

How to run PMM Server with Docker based on our [Docker image].

!!! note alert alert-primary ""
    The tags used here are for the current release. Other [tags] are available.

!!! seealso alert alert-info "See also"
    [Easy-install script]

## Before you start

- Install [Docker] 1.12.6 or higher.
- For PMM 2.38.0 or greater, ensure your CPU (and any virtualization layer you may be using) supports `x86-64-v2`

## Run

!!! summary alert alert-info "Summary"
    - Pull the Docker image.
    - Copy it to create a persistent data container.
    - Run the image.
    - Open the PMM UI in a browser.

---

You can store data from your PMM in:

1. Docker volume (Preffered method)
2. Data container
3. Host directory


### Run Docker with volume

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Create a volume:

    ```sh
    docker volume create pmm-data
    ```

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    -v pmm-data:/srv \
    --name pmm-server \
    percona/pmm-server:2
    ```
    
4. Change the password for the default `admin` user.

    * For PMM versions 2.27.0 and later:

    ```sh
    docker exec -t pmm-server change-admin-password <new_password>
    ```

    * For PMM versions prior to 2.27.0:

    ```sh
    docker exec -t pmm-server bash -c 'grafana-cli --homepath /usr/share/grafana --configOverrides cfg:default.paths.data=/srv/grafana admin reset-admin-password newpass'
    ```

5. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)
       

### Run Docker with data container



1. Create a persistent data container.

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

2. Run the image.

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    --volumes-from pmm-data \
    --name pmm-server \
    percona/pmm-server:2
    ```

3. Change the password for the default `admin` user.

    * For PMM versions 2.27.0 and later:

    ```sh
    docker exec -t pmm-server change-admin-password <new_password>
    ```

    * For PMM versions prior to 2.27.0:

        ```sh
        docker exec -t pmm-server bash -c 'grafana-cli --homepath /usr/share/grafana --configOverrides cfg:default.paths.data=/srv/grafana admin reset-admin-password newpass'
        ```
        
4. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

### Run Docker with the host directory

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Run the image.

    ```sh
    export DATA_DIR=$HOME/srv
    docker run -v $DATA_DIR/srv:/srv -d --restart always --publish 80:80 --publish 443:443 --name pmm-server percona/pmm-server:2
    ```
    `DATA_DIR` is a directory where you want to store the state for PMM.


3. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

### Migrate from data container to host directory/volume

To migrate your PMM from data container to host directory or volume run the following command:
```sh
docker cp <containerId>:/srv /target/host/directory
```



## Backup

!!! summary alert alert-info "Summary"
    - Stop and rename the `pmm-server` container.
    - Take a local copy of the `pmm-data` container's `/srv` directory.

---

!!! caution alert alert-warning "Important"
    Grafana plugins have been moved to the data volume `/srv` since the 2.23.0 version. So if you are upgrading PMM from any version before 2.23.0 and have installed additional plugins then plugins should be installed again after the upgrade.
    
    To check used grafana plugins:

    ```sh
    docker exec -it pmm-server ls /var/lib/grafana/plugins
    ```

1. Stop the container.

    ```sh
    docker stop pmm-server
    ```

2. Move the image.

    ```sh
    docker rename pmm-server pmm-server-backup
    ```

3. Create a subdirectory (e.g., `pmm-data-backup`) and move to it.

    ```sh
    mkdir pmm-data-backup && cd pmm-data-backup
    ```

4. Backup the data.

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

!!! caution alert alert-warning "Important"
    Downgrades are not possible. To go back to using a previous version you must have created a backup of it before upgrading.

!!! hint alert alert-success "Tip"
    To see what release you are running, use the *PMM Upgrade* panel on the *Home Dashboard*, or run:

    ```sh
    docker exec -it pmm-server \
    curl -ku admin:admin https://localhost/v1/version
    ```

    (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)


1. Stop the container.

    ```sh
    docker stop pmm-server
    ```

2. Perform a [backup](#backup).


3. Pull the latest image.

    ```sh
    docker pull percona/pmm-server:2
    ```

4. Rename the original container

    ```sh
    docker rename pmm-server pmm-server-old
    ```


5. Run it.

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

2. Remove it.

    ```sh
    docker rm pmm-server
    ```

3. Revert to the saved image.

    ```sh
    docker rename pmm-server-backup pmm-server
    ```

4. Change directory to the backup directory (e.g. `pmm-data-backup`).

5. Remove Victoria Metrics data folder.

    ```sh
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 rm -r /srv/victoriametrics/data
    ```

6. Copy the data.

    ```sh
    docker cp srv pmm-data:/
    ```

7. Restore permissions.

    ```sh
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:root /srv && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/alertmanager && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R root:pmm /srv/clickhouse && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R grafana:grafana /srv/grafana && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/logs && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/postgres14 && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/prometheus && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R pmm:pmm /srv/victoriametrics && \
    docker run --rm --volumes-from pmm-data -it percona/pmm-server:2 chown -R postgres:postgres /srv/logs/postgresql14.log
    ```

8. Start the image.

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

2. Remove containers.

    ```sh
    docker rm pmm-server pmm-data
    ```

3. Remove the image.

    ```sh
    docker rmi $(docker images | grep "percona/pmm-server" | awk {'print $3'})
    ```


## Environment variables

Use the following Docker container environment variables (with `-e var=value`) to set PMM Server parameters.

| Variable  &nbsp; &nbsp; &nbsp; &nbsp;                              | Description
| --------------------------------------------------------------- | -----------------------------------------------------------------------
| `DISABLE_UPDATES`                                               | Disables a periodic check for new PMM versions as well as ability to apply upgrades using the UI
| `DISABLE_TELEMETRY`                                             | Disable built-in telemetry and disable STT if telemetry is disabled.
| `METRICS_RESOLUTION`                                            | High metrics resolution in seconds.
| `METRICS_RESOLUTION_HR`                                         | High metrics resolution (same as above).
| `METRICS_RESOLUTION_MR`                                         | Medium metrics resolution in seconds.
| `METRICS_RESOLUTION_LR`                                         | Low metrics resolution in seconds.
| `DATA_RETENTION`                                                | The number of days to keep time-series data. <br />**N.B.** This must be set in a format supported by `time.ParseDuration` <br /> and represent the complete number of days. <br /> The supported units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, and `h`. <br /> The value must be a multiple of 24, e.g., for 90 days 2160h (90 * 24).
| `ENABLE_VM_CACHE`                                               | Enable cache in VM.
| `DISABLE_ALERTING`                           | Disables built-in Percona Alerting, which is enabled by default.
| `ENABLE_AZUREDISCOVER`                                          | Enable support for discovery of Azure databases.
| `DISABLE_BACKUP_MANAGEMENT`                                     | Disables Backup Management, which is enabled by default.
| `ENABLE_DBAAS`                                                  | Enable DBaaS features.
| `PMM_DEBUG`                                                     | Enables a more verbose log level.
| `PMM_TRACE`                                                     | Enables a more verbose log level including trace-back information.
| `PMM_PUBLIC_ADDRESS`                                            | External IP address or the DNS name on which PMM server is running.

The following variables are also supported but values passed are not verified by PMM. If any other variable is found, it will be considered invalid and the server won't start.

| Variable                                                        | Description
| --------------------------------------------------------------- | ------------------------------------------------------
| `_`, `HOME`, `HOSTNAME`, `LANG`, `PATH`, `PWD`, `SHLVL`, `TERM` | Default environment variables.
| `GF_*`                                                          | [Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/) environment variables.
| `VM_*`                                                          | [VictoriaMetrics'](https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#environment-variables) environment variables. 
Note that environment variables inherit their names from the command line flags. To find out which variables are available to you, see the full list of [CLI command flags](https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#list-of-command-line-flags). 

| `SUPERVISOR_`                                                   | `supervisord` environment variables.
| `KUBERNETES_`                                                   | Kubernetes environment variables.
| `MONITORING_`                                                   | Kubernetes monitoring environment variables.
| `PERCONA_TEST_`                                                 | Unknown variable but won't prevent the server starting.
| `PERCONA_TEST_DBAAS`                                            | Deprecated. Use `ENABLE_DBAAS`.


## Preview environment variables

!!! caution alert alert-warning "Warning"
     The `PERCONA_TEST_*` environment variables are experimental and subject to change. It is recommended that you use these variables for testing purposes only and not on production.

| Variable                                                                   | Description
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------
| `PERCONA_TEST_SAAS_HOST`                                                   | SaaS server hostname.
| `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`                                         | Name of the host and port of the external ClickHouse database instance.
| `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE`                                     | Database name of the external ClickHouse database instance.
| `​​PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`                                    | The maximum number of threads in the current connection thread pool. This value cannot be bigger than max_thread_pool_size.
| `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE`                                   | The number of rows to load from tables in one block for this connection.


## Tips

- To Disable the Home Dashboard *PMM Upgrade* panel you can either add `-e DISABLE_UPDATES=true` to the `docker run` command (for the life of the container) or navigate to _PMM --> PMM Settings --> Advanced Settings_ and disable "Check for Updates" (can be turned back on by any admin in the UI).

- Eliminate browser certificate warnings by configuring a [trusted certificate].

- You can optionally enable an (insecure) HTTP connection by adding `--publish 80:80` to the `docker run` command. However, running PMM insecure is not recommended. You should also note that PMM Client *requires* TLS to communicate with the server, only working on a secure port.

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

6. [Run the container](#run) as if your image is already pulled using your desired method for a storage volume (you can step over any docker pull commands as the image has been pre-staged).

[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Docker]: https://docs.docker.com/get-docker/
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker compose]: https://docs.docker.com/compose/
[PMMC_COMPOSE]: ../client/index.md#docker-compose
[trusted certificate]: ../../how-to/secure.md#ssl-encryption
[Easy-install script]: easy-install.md

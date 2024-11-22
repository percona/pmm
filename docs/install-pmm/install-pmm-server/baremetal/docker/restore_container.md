# Restore container

??? info "Summary"

    !!! summary alert alert-info ""
        - Stop and remove the container.
        - Restore (rename) the backup container.
        - Restore saved data to the data container.
        - Restore permissions to the data.

    ---

!!! caution alert alert-warning "Important"
    You must have a [backup](backup_container.md) to restore from.

To restore the container:
{.power-number}

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
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta rm -r /srv/victoriametrics/data
    ```

6. Copy the data.

    ```sh
    docker cp srv pmm-data:/
    ```

7. Restore permissions.

    ```sh
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R root:root /srv && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R pmm:pmm /srv/alertmanager && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R root:pmm /srv/clickhouse && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R grafana:grafana /srv/grafana && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R pmm:pmm /srv/logs && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R postgres:postgres /srv/postgres14 && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R pmm:pmm /srv/prometheus && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R pmm:pmm /srv/victoriametrics && \
    docker run --rm --volumes-from pmm-data -it perconalab/pmm-server:3.0.0-beta chown -R postgres:postgres /srv/logs/postgresql14.log
    ```

8. Start the image.

    ```sh
    docker start pmm-server
    ```



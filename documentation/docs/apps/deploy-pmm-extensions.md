# Set up Apps: Deploy PMM Extensions with Docker

!!! warning "This feature is in Tech Preview"
    This feature is not yet production-ready. Use it only for testing and feedback.

Deploy PMM Extensions to enable [database management apps](index.md) in PMM. These apps enable you to run operations on your database hosts directly from PMM, without SSH access.

## Limitations

You cannot deploy PMM Extensions on:

- AMI deployments, because PMM Extensions is only available as a Docker container.
- A separate host from PMM Server, because traffic between the two containers is not encrypted.
- arm64 or other non-amd64 architectures.
- [PMM HA](../install-pmm/HA.md) or external PostgreSQL deployments.

## Check versions before deploying

PMM Server and PMM Extensions must run on the same version. If they don't match, the app pages show a warning with both versions.

Both containers use a single version tag, `PMM_IMAGE_TAG`. The default is `3`, which pulls the latest 3.x release. To run a specific version, set it in `.env` before starting the containers:

```ini
PMM_IMAGE_TAG=3.10.0
```

If your environment pulls images from a registry other than Docker Hub, change the two `image:` lines in the `docker-compose.yml` file on this page to point to your registry. Use the same version for both images.

## Deploy PMM Extensions

=== "Existing PMM install"

    You can't add PMM Extensions to a running PMM container, because PMM applies its configuration at startup. Instead, recreate the `pmm-server` container with the new settings. Your existing data volume stays as it is, and the only downtime is a single PMM restart:
    {.power-number}

    1. Add these environment variables to your `pmm-server` container:

        ```ini
        PMM_ENABLE_EXTENSIONS=1
        PMM_ENABLE_NOMAD=1
        PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
        ```

    2. Expose port `4647` on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes on other hosts connect to this port to receive and run tasks. Without it, only tasks targeting the PMM Server host itself will work.

    3. Add an empty `pmm-extensions` volume to the `pmm-server` container and mount it at `/srv/extensions`. PMM uses this volume to share the credentials that PMM Extensions needs to start. Make sure the volume is empty before starting the container.

    4. Recreate the `pmm-server` container to apply the new settings. Your data in the `pmm-data` volume stays intact.

    5. Start the PMM Extensions container:

        === "Docker Compose"

            Remove the existing `pmm-server` container and let Compose take over. Compose manages its own `pmm-server` and will conflict with an existing one on container name and port.
            {.power-number}

            1. Stop and remove the existing container. Your data in `pmm-data` is preserved:

                ```bash
                docker stop pmm-server && docker rm pmm-server
                ```

            2. Create a `docker-compose.yml` file with both services, then set `PMM_PUBLIC_ADDRESS` in `.env` and run `docker compose up -d`:

                ```yaml
                services:
                  pmm-server:
                    image: percona/pmm-server:${PMM_IMAGE_TAG:-3}
                    container_name: pmm-server
                    restart: always
                    ports:
                      - "443:8443"
                      - "4647:4647"
                    environment:
                      PMM_ENABLE_EXTENSIONS: "1"
                      PMM_ENABLE_NOMAD: "1"
                      PMM_PUBLIC_ADDRESS: ${PMM_PUBLIC_ADDRESS}
                    volumes:
                      - pmm-data:/srv
                      - pmm-extensions:/srv/extensions

                  pmm-extensions:
                    image: percona/pmm-extensions:${PMM_IMAGE_TAG:-3}
                    container_name: pmm-extensions
                    restart: always
                    group_add:
                      - "0"
                    environment:
                      SECRETS_DIR: /run/secrets/extensions
                      GF_SECURITY_ADMIN_USER: admin
                      GF_SECURITY_ADMIN_PASSWORD: ${PMM_ADMIN_PASSWORD:-admin}
                      BASE_URL: https://${PMM_PUBLIC_ADDRESS}/extensions
                    volumes:
                      - pmm-extensions:/run/secrets/extensions:ro
                      - pmm-extensions-state:/home/extensions/state
                    depends_on:
                      - pmm-server

                volumes:
                  pmm-data: {}
                  pmm-extensions: {}
                  pmm-extensions-state: {}
                ```

                Add any other `pmm-server` settings you had before (custom ports, TLS certificates, extra environment variables) to the `pmm-server` service.

        === "docker run"

            PMM Server routes `/extensions/` requests to `pmm-extensions:9000` by container name. Both containers must be on the same user-defined Docker network, or they can't reach each other by name and the **Apps** menu will not appear.
            {.power-number}

            1. If `pmm-server` is not already on a user-defined network, create one and attach it:

                ```bash
                docker network create pmm-net
                docker network connect pmm-net pmm-server
                ```

            2. Start the PMM Extensions container on the same network, using exactly `--name pmm-extensions`:

                ```bash
                docker run -d \
                  --name pmm-extensions \
                  --restart always \
                  --network pmm-net \
                  --group-add 0 \
                  -e SECRETS_DIR=/run/secrets/extensions \
                  -e GF_SECURITY_ADMIN_USER=admin \
                  -e GF_SECURITY_ADMIN_PASSWORD=admin \
                  -e BASE_URL=https://<PMM_PUBLIC_ADDRESS>/extensions \
                  -v pmm-extensions:/run/secrets/extensions:ro \
                  -v pmm-extensions-state:/home/extensions/state \
                  percona/pmm-extensions:3.10.0
                ```

    6. If you changed the default PMM admin password, set `GF_SECURITY_ADMIN_PASSWORD` to match. PMM Extensions uses it to create its service account token. If the value is wrong, the container starts, but the apps can't connect to PMM, and `docker logs pmm-extensions` shows the error.

=== "New PMM install"

    Use this option if you are setting up PMM for the first time and want to include PMM Extensions from the start. Create a `docker-compose.yml` file with both services and a `.env` file with your settings:
    {.power-number}

    1. Create an `.env` file with your settings. `PMM_PUBLIC_ADDRESS` is required. If you skip it, PMM Extensions starts but can't run tasks and won't show an error:

        ```ini
        # To pin a specific version (default pulls the latest 3.x release):
        # PMM_IMAGE_TAG=3.10.0
        PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
        ```

    2. Create a `docker-compose.yml` file with the following content:

        ```yaml
        services:
          pmm-server:
            image: percona/pmm-server:${PMM_IMAGE_TAG:-3}
            container_name: pmm-server
            restart: always
            ports:
              - "443:8443"
              - "4647:4647"
            environment:
              PMM_ENABLE_EXTENSIONS: "1"
              PMM_ENABLE_NOMAD: "1"
              PMM_PUBLIC_ADDRESS: ${PMM_PUBLIC_ADDRESS}
            volumes:
              - pmm-data:/srv
              - pmm-extensions:/srv/extensions

          pmm-extensions:
            image: percona/pmm-extensions:${PMM_IMAGE_TAG:-3}
            container_name: pmm-extensions
            restart: always
            group_add:
              - "0"
            environment:
              SECRETS_DIR: /run/secrets/extensions
              GF_SECURITY_ADMIN_USER: admin
              GF_SECURITY_ADMIN_PASSWORD: ${PMM_ADMIN_PASSWORD:-admin}
              BASE_URL: https://${PMM_PUBLIC_ADDRESS}/extensions
            volumes:
              - pmm-extensions:/run/secrets/extensions:ro
              - pmm-extensions-state:/home/extensions/state
            depends_on:
              - pmm-server

        volumes:
          pmm-data: {}
          pmm-extensions: {}
          pmm-extensions-state: {}
        ```

    3. Start PMM Server and PMM Extensions together:

        ```bash
        docker compose up -d
        ```

## Verify the deployment

After the containers start, confirm that PMM Extensions is running and integrated with PMM:
{.power-number}

1. Check that both containers are running and healthy:

    ```bash
    docker ps
    ```

2. Check that PMM Extensions responds. A `200` response means it is running:

    ```bash
    curl -sk -o /dev/null -w "%{http_code}\n" https://<PMM_SERVER_ADDRESS>/extensions/health
    ```

3. Open PMM and check that the **Apps** menu appears in the sidebar, with **MySQL Backups** and **Support diagnostics** listed under it.

## Back up the PMM Extensions state volume

The `pmm-extensions-state` volume holds the encryption key for PMM Extensions settings stored in `pmm-data`. Without it, a `pmm-data` backup is unusable, so back up both volumes together whenever you [back up the PMM Server container](../install-pmm/install-pmm-server/deployment-options/docker/backup_container.md).

## Disable PMM Extensions

Disabling PMM Extensions doesn't delete its data. PMM keeps the `pmm_extensions` database and its role on purpose, so you can turn PMM Extensions back on later without losing data.

To disable PMM Extensions:
{.power-number}

1. Unset `PMM_ENABLE_EXTENSIONS` and recreate the `pmm-server` container.

2. Stop and remove the PMM Extensions container. Use the command that matches how you started it:

    - **docker run**: `docker stop pmm-extensions && docker rm pmm-extensions`
    - **Docker Compose**: `docker compose rm -s -f pmm-extensions`, then delete the `pmm-extensions` service from your `docker-compose.yml`

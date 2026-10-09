# Set up Apps: Deploy PMM Extensions with Docker

!!! warning "This feature is in Tech Preview"
    This feature is not yet production-ready. Use it only for testing and feedback.

Deploy PMM Extensions to enable [database management apps](index.md) in PMM. These apps let you run operations on your database hosts directly from PMM, without SSH access.

## Limitations

You cannot deploy PMM Extensions on:

- AMI deployments, because PMM Extensions is only available as a Docker container.
- A separate host from PMM Server, because traffic between the two containers is not encrypted.
- arm64 or other non-amd64 architectures.
- [PMM HA](../install-pmm/HA.md) or external PostgreSQL deployments.

## Check versions before deploying

PMM Server and PMM Extensions must run on the same version. A mismatch won't produce an error but the **Apps** menu won't appear in the sidebar.

Both containers use a single version tag, `PMM_IMAGE_TAG`. The default is `3`, which pulls the latest 3.x release. To run a specific version, set it in `.env` before starting the containers:

```ini
PMM_IMAGE_TAG=3.10.0
```

If your environment requires pulling images from a location other than Docker Hub, use `PMM_SERVER_IMAGE` and `PMM_EXTENSIONS_IMAGE` to specify the full image path for each container. These replace `PMM_IMAGE_TAG`, so set both to the same version.

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

    2. Add an empty `pmm-extensions` volume to the `pmm-server` container and mount it at `/srv/extensions`. PMM uses this volume to share the credentials that PMM Extensions needs to start. Make sure the volume is empty before starting the container.

    3. Recreate the `pmm-server` container to apply the new settings. Your data in the `pmm-data` volume stays intact.

    4. Start the PMM Extensions container:

        === "Docker Compose"

            Remove the existing `pmm-server` container and let Compose take over. Compose manages its own `pmm-server` and will conflict with an existing one on container name and port.
            {.power-number}

            1. Stop and remove the existing container. Your data in `pmm-data` is preserved:

                ```bash
                docker stop pmm-server && docker rm pmm-server
                ```

            2. Start both containers. Both profiles are required: `--profile extensions` alone fails and creates nothing:

                ```bash
                docker compose --profile pmm --profile extensions up -d
                ```

        === "docker run"

            Once `pmm-server` is running with the updated configuration from step 3, start the PMM Extensions container on the same network:

            ```bash
            docker run -d \
              --name pmm-extensions \
              --restart always \
              --network <same-network-as-pmm-server> \
              --group-add 0 \
              -e SECRETS_DIR=/run/secrets/extensions \
              -e GF_SECURITY_ADMIN_USER=admin \
              -e GF_SECURITY_ADMIN_PASSWORD=admin \
              -e BASE_URL=https://<PMM_PUBLIC_ADDRESS>/extensions \
              -v pmm-extensions:/run/secrets/extensions:ro \
              -v pmm-extensions-state:/home/extensions/state \
              percona/pmm-extensions:3.10.0
            ```

    5. (Optional) If you changed the default PMM admin password, update `GF_SECURITY_ADMIN_PASSWORD` to match. If the value is wrong, the container starts but sign-in and PMM inventory sync fail without an error.

=== "New PMM install"

    Use this option if you are setting up PMM for the first time and want to include PMM Extensions from the start. The `docker-compose.yml` file in the PMM repository includes PMM Extensions under the `extensions` profile, so you can start both containers with a single command:
    {.power-number}

    1. Copy `.env.example` to `.env` and set these variables. If you skip `PMM_ENABLE_NOMAD` or `PMM_PUBLIC_ADDRESS`, PMM Extensions starts but can't run tasks and won't show an error:

        ```ini
        # To pin a specific version (default pulls the latest 3.x release):
        # PMM_IMAGE_TAG=3.10.0
        PMM_ENABLE_EXTENSIONS=1
        PMM_ENABLE_NOMAD=1
        PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
        # Only if PMM Client nodes on other hosts will run app tasks:
        PMM_EXTENSIONS_BASE_URL=https://<PMM_PUBLIC_ADDRESS>/extensions
        ```

    2. Start PMM Server and PMM Extensions together:

        ```bash
        docker compose --profile pmm --profile extensions up -d
        ```

## Verify the deployment

After the containers start, confirm that PMM Extensions is running and integrated with PMM:
{.power-number}

1. Check that both containers are running and healthy:

    ```bash
    docker ps
    ```

2. Check that PMM Extensions is running. If the command returns `200`, PMM Extensions is running:

    ```bash
    curl -sk -o /dev/null -w "%{http_code}\n" https://<PMM_SERVER_ADDRESS>/extensions/health
    ```

3. Open PMM and check that the **Apps** menu appears in the sidebar, with **MySQL Backups** and **Support Diagnostics** listed under it.

## Back up the PMM Extensions state volume

The `pmm-extensions-state` volume holds the encryption key for PMM Extensions settings stored in `pmm-data`. Without it, a `pmm-data` backup is unusable, so back up both volumes together whenever you [back up the PMM Server container](../install-pmm/install-pmm-server/deployment-options/docker/backup_container.md).

## Disable PMM Extensions

Disabling PMM Extensions doesn't delete its data. PMM keeps the `pmm_extensions` database and its role on purpose, so you can turn PMM Extensions back on later without losing data.

To disable PMM Extensions:
{.power-number}

1. Unset `PMM_ENABLE_EXTENSIONS` and recreate the `pmm-server` container.

2. Stop and remove the PMM Extensions container:

    ```bash
    docker compose --profile extensions down
    ```

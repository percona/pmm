# Deploy PMM Extensions with Docker

!!! warning "Tech Preview"
    This feature is not production-ready. Use for testing and feedback only.

Deploy PMM Extensions to enable [database management apps](index.md) in PMM. Apps let you run operations on your database hosts directly from PMM, without SSH access.

## Limitations

PMM Extensions cannot be deployed on:

- AMI deployments as it is only available as a Docker container.
- a separate host from PMM Server because traffic between the two containers is not encrypted.
- arm64 or other non-amd64 architectures. Only amd64 binaries are available.
- [PMM HA](../install-pmm/HA.md) or [external PostgreSQL](../install-pmm/install-pmm-server/deployment-options/docker/docker.md) deployments since PMM Extensions relies on PMM's built-in PostgreSQL database.

## Before you start

Make sure your PMM Server and PMM Extensions images run the same version. If they don't match, the **Apps** pages will be missing from the sidebar but no error is displayed.

## Scenario 1: Add PMM Extensions to an existing install

PMM Extensions cannot be added to a running PMM container. PMM sets everything up at startup, so you need to recreate the pmm-server container with the new settings. Downtime is one PMM restart.
{.power-number}

1. Add these environment variables to your pmm-server container:

    ```ini
    PMM_ENABLE_EXTENSIONS=1
    PMM_ENABLE_NOMAD=1
    PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
    ```

2. Add a new, empty `pmm-extensions` volume mounted at `/srv/extensions` on the pmm-server container. The volume must be empty before PMM Server starts: PMM writes credentials into it at startup, and PMM Extensions reads them on launch. If the volume already contains files, PMM Extensions exits.

3. Recreate pmm-server. Your existing data volume (`pmm-data`) is unaffected.

4. Once pmm-server is healthy, start the PMM Extensions container. Use the `pmm-extensions` service definition from `docker-compose.yml` as reference:

    === "Docker Compose"
    
        ```bash
        docker compose --profile extensions up -d
        ```

    === "docker run"

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

    If you have changed the PMM admin password from the default, pass it as `GF_SECURITY_ADMIN_PASSWORD`. PMM Extensions uses it to create its own access token on first start. If the password is wrong, PMM Extensions starts but sign-in and inventory sync silently fail.

## Scenario 2: New Docker Compose install

The `docker-compose.yml` in the PMM repository ships PMM Extensions under the `extensions` profile.
{.power-number}

1. Copy `.env.example` to `.env` and set:

    ```ini
    PMM_ENABLE_EXTENSIONS=1
    PMM_ENABLE_NOMAD=1
    PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
    ```

    Both `PMM_ENABLE_NOMAD=1` and `PMM_PUBLIC_ADDRESS` are required. If either is missing, Nomad does not start, and no error is reported.

    If PMM Client nodes on other hosts will run Apps tasks, also set:

    ```ini
    PMM_EXTENSIONS_BASE_URL=https://<PMM_PUBLIC_ADDRESS>/extensions
    ```

2. Start PMM Server and PMM Extensions together:

    ```bash
    docker compose --profile pmm --profile extensions up -d
    ```

## Verify the deployment

1. Check that both containers are running and healthy:

    ```bash
    docker ps
    ```

2. Check the PMM Extensions health endpoint:

    ```bash
    curl -sk https://<PMM_SERVER_ADDRESS>/extensions/health
    ```

    A `200` response confirms PMM Extensions is running.

3. Open PMM. **MySQL Backups** and **Support Diagnostics** should appear under **Apps** in the sidebar.

## Back up the PMM Extensions state volume

PMM Extensions stores its access token and the encryption key for its settings in the `pmm-extensions-state` volume. Back it up alongside your `pmm-data` volume. If the state volume is lost, PMM Extensions cannot read its saved settings.

## Disable PMM Extensions

1. Unset `PMM_ENABLE_EXTENSIONS` and recreate pmm-server.

2. Stop and remove the PMM Extensions container:

    ```bash
    docker compose --profile extensions down
    ```

The `pmm_extensions` database and its role are kept intentionally, so re-enabling PMM Extensions later preserves all saved data.

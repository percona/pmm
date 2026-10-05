# Deploy PMM Extensions with Docker

!!! warning "Tech Preview"
    This feature is not yet production-ready. Use it only for testing and feedback.

Deploy PMM Extensions to enable [database management apps](index.md) in PMM. These apps let you run operations on your database hosts directly from PMM, without SSH access.

## Limitations

You cannot deploy PMM Extensions on:

- AMI deployments, because PMM Extensions is only available as a Docker container.
- A separate host from PMM Server, because traffic between the two containers is not encrypted.
- arm64 or other non-amd64 architectures.
- [PMM HA](../install-pmm/HA.md) or [external PostgreSQL](../install-pmm/install-pmm-server/deployment-options/docker/docker.md) deployments.

## Before you start

Make sure the PMM Server and PMM Extensions images use the same version. If the versions don’t match, the **Apps** menu won’t appear in the sidebar, and PMM won’t display an error.

## Scenario 1: Add PMM Extensions to an existing install

You can't add PMM Extensions to a running PMM container, because PMM applies its configuration at startup. Instead, recreate the `pmm-server` container with the new settings. Your existing data volume stays as it is, and the only downtime is a single PMM restart:
{.power-number}

1. Add these environment variables to your `pmm-server` container:

    ```ini
    PMM_ENABLE_EXTENSIONS=1
    PMM_ENABLE_NOMAD=1
    PMM_PUBLIC_ADDRESS=<address PMM Clients use to reach this server>
    ```

2. Add an empty `pmm-extensions` volume to the `pmm-server` container and mount it at `/srv/extensions`.
PMM uses this volume to share the credentials that PMM Extensions needs to start. Make sure the volume is empty before starting the container.

3. Recreate the `pmm-server` container to apply the new settings. Your data in the `pmm-data` volume stays intact.

4. Once `pmm-server` is healthy, start the PMM Extensions container. Use the `pmm-extensions` service definition from `docker-compose.yml` as reference:

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

    If you changed the default PMM admin password, pass the new one to the PMM Extensions container as `GF_SECURITY_ADMIN_PASSWORD`. PMM Extensions uses it to create its own access token. If the password is wrong, the container still starts, but sign-in and PMM inventory sync fail without any error.
    
## Scenario 2: New Docker Compose install
Use this option for a new installation. The `docker-compose.yml` file in the PMM repository includes PMM Extensions under the `extensions` profile, so you can start both containers with a single command:
{.power-number}

1. Copy `.env.example` to `.env` and set these variables. `PMM_ENABLE_NOMAD` and `PMM_PUBLIC_ADDRESS` are both required: without them, PMM Extensions can't run tasks, and no error is shown. Set `PMM_EXTENSIONS_BASE_URL` only if PMM Client nodes on other hosts will run app tasks:

    ```ini
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

PMM Extensions stores its settings in PMM's built-in PostgreSQL database, inside `pmm-data`, but keeps the key that decrypts them in `pmm-extensions-state`. 

A `pmm-data` backup is only usable if you also have the matching key, so back up `pmm-extensions-state` whenever you [back up `pmm-data`](../install-pmm/install-pmm-server/deployment-options/docker/backup_container.md).

## Disable PMM Extensions

Disabling PMM Extensions doesn't delete its data. PMM keeps the `pmm_extensions` database and its role on purpose, so you can turn PMM Extensions back on later without losing data.

To disable PMM Extensions:
{.power-number}

1. Unset `PMM_ENABLE_EXTENSIONS` and recreate the `pmm-server` container.

2. Stop and remove the PMM Extensions container:

```bash
docker compose --profile extensions down
```
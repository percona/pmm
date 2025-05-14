# Migrate external PostgreSQL configuration for PMM 3.2.0+ upgrades

If you're using an external PostgreSQL database with PMM, you will need to update your configuration before upgrading to PMM 3.2.0 or later. This is because PMM 3.2.0 includes Grafana 11.6, which introduces a [regression issue](https://github.com/grafana/grafana/issues/102337): the single `GF_DATABASE_URL` environment variable is no longer sufficient for database configuration.

To upgrade to PMM 3.2.0 successfully, convert your configuration to use individual environment variables instead of `GF_DATABASE_URL`: 

## Before you begin

- Verify you're using an external PostgreSQL database with PMM
- Ensure you have the connection details for your PostgreSQL database
- Back up your PMM data before proceeding

## Migration procedure

### 1. Determine your current database URL format

Currently, your external PostgreSQL is likely configured with a connection string like: `GF_DATABASE_URL=postgres://USER:PASSWORD@HOST:PORT/DATABASE_NAME`. 

You'll need to extract these components for the new configuration format.

### 2. Stop your PMM Server
Depending on your deployment method, use one of the following approaches:

=== "Docker"
    ```bash
    docker stop pmm-server
    docker rm pmm-server
    ```

=== "Podman"
    ```bash
    podman stop pmm-server
    podman rm pmm-server
    ```

=== "Docker Compose"
    ```bash
    docker-compose down
    ```

=== "Kubernetes/Helm"
    ```bash
    # Scale down the PMM deployment
    kubectl scale deployment pmm-server --replicas=0 -n <namespace>
    # Wait for the pod to terminate
    kubectl wait --for=delete pod -l app=pmm-server -n <namespace>
    ```

### 3. Replace the database URL with individual parameters

Convert your database URL to the following individual environment variables:

| Old Format | New Format |
|------------|------------|
| `GF_DATABASE_URL=postgres://USER:PASSWORD@HOST:PORT/DATABASE_NAME` | `GF_DATABASE_TYPE=postgresql`<br>`GF_DATABASE_USER=USER`<br>`GF_DATABASE_PASSWORD=PASSWORD`<br>`GF_DATABASE_HOST=HOST:PORT`<br>`GF_DATABASE_NAME=DATABASE_NAME` |

!!! note
    If your database URL doesn't specify a port, the default PostgreSQL port 5432 will be used.

### 4. Restart PMM Server with the new configuration

Modify your startup command or configuration file to use the new parameters:

=== "Docker"
    ```bash
    docker run -d \
      -p 443:443 \
      -v pmm-data:/srv \
      -e GF_DATABASE_TYPE=postgresql \
      -e GF_DATABASE_USER=your_user \
      -e GF_DATABASE_PASSWORD=your_password \
      -e GF_DATABASE_HOST=your_host:your_port \
      -e GF_DATABASE_NAME=your_db_name \
      -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
      --name pmm-server \
      percona/pmm-server:3.2.0
    ```

=== "Podman"
    ```bash
    podman run -d \
      -p 443:443 \
      -v pmm-data:/srv \
      -e GF_DATABASE_TYPE=postgresql \
      -e GF_DATABASE_USER=your_user \
      -e GF_DATABASE_PASSWORD=your_password \
      -e GF_DATABASE_HOST=your_host:your_port \
      -e GF_DATABASE_NAME=your_db_name \
      -e PMM_DISABLE_BUILTIN_POSTGRES=1 \
      --name pmm-server \
      percona/pmm-server:3.2.0
    ```

=== "Docker Compose"
    Update your `docker-compose.yml` file:
    ```yaml
    services:
      pmm-server:
        image: percona/pmm-server:3.2.0
        ports:
          - "443:443"
        volumes:
          - pmm-data:/srv
        environment:
          - GF_DATABASE_TYPE=postgresql
          - GF_DATABASE_USER=your_user
          - GF_DATABASE_PASSWORD=your_password
          - GF_DATABASE_HOST=your_host:your_port
          - GF_DATABASE_NAME=your_db_name
          - PMM_DISABLE_BUILTIN_POSTGRES=1
        restart: always
    
    volumes:
      pmm-data:
    ```
    Then restart:
    ```bash
    docker-compose up -d
    ```

=== "Kubernetes/Helm"
    Update your values file to include the new parameters:
    ```yaml
    env:
      - name: GF_DATABASE_TYPE
        value: postgresql
      - name: GF_DATABASE_USER
        value: your_user
      - name: GF_DATABASE_PASSWORD
        value: your_password
      - name: GF_DATABASE_HOST
        value: your_host:your_port
      - name: GF_DATABASE_NAME
        value: your_db_name
      - name: PMM_DISABLE_BUILTIN_POSTGRES
        value: "1"
    ```
    Then upgrade or restart:
    ```bash
    helm upgrade pmm percona/pmm-server -n <namespace> -f values.yaml
    # Or scale back up if you scaled down earlier
    kubectl scale deployment pmm-server --replicas=1 -n <namespace>
    ```

### 5. Verify the upgrade
After completing the upgrade, follow these steps to ensure PMM Server is functioning correctly and your external PostgreSQL database is properly connected:
{.power-number}

1. Wait for PMM Server to start
2. Access the PMM web interface
3. Check that dashboards and metrics are loading correctly
4. Verify that no database connection errors appear in the PMM Server logs

## Troubleshooting

If PMM fails to connect to your PostgreSQL database, verify that:
{.power-number}

1. Database credentials are correct
2. Database host is accessible from the PMM Server container
3. PostgreSQL is configured to accept connections from PMM Server's IP address
4. PostgreSQL server is running and healthy

You can check the Grafana logs for more detailed error messages: `docker exec pmm-server grep -i database /srv/logs/grafana.log`. 
# Migrate PMM 2 to PMM 3

If you are still running PMM 2, plan your migration now. PMM 2 reached end of life on October 31, 2025 and no longer receives new features, security updates, or bug fixes.

Starting with PMM 3.8.0, direct migration from PMM 2.x to the latest PMM 3.x version will be deprecated and may not work as expected. If you migrate after PMM 3.8.0 and run into issues, you can still use PMM 3.7 as a stepping stone since this is the last version where migration from PMM 2.x has been fully tested.

This two-step path will only be available through PMM 3.12.0 After PMM 3.13.0 (expected January 2027), you will no longer be able to migrate from PMM 2.x at all.

## Migration path

To migrate from PMM 2.x, you can try upgrading directly to the latest PMM 3.x version. If the migration fails, use PMM 3.7.0 as a fallback since this is the last version where migration from PMM 2.x has been fully tested:

```
PMM 2.x > PMM 2.44.1 > PMM 3.7 > latest PMM 3.x
```

## Step 1: Create a backup before any changes

Migration involves significant architectural changes that cannot be reversed without a backup. 
Before proceeding with migration, you must [create a complete backup](../install-pmm/install-pmm-server/deployment-options/docker/backup_container.md) of your current PMM 2 deployment. 

## Step 2: Upgrade to the latest PMM 2 release

Before migrating to PMM 3, ensure your PMM 2 Server is running PMM 2.44.1 (the final PMM 2 release). Migration is only tested and supported from this version.

=== "From the UI"
    Use the built-in upgrade feature in PMM 2 to update to the latest version:
    {.power-number}

    1. From the **Home** page, scroll to the **PMM Upgrade** panel and click **Refresh** to check for updates.
    2. If an update is available, click **Update** to install PMM 2.44.1.
    3. Verify the version number shows 2.44.1.

=== "Using Docker"
    If your PMM 2 Server runs in Docker, pull the latest PMM 2 image and recreate the container:
    {.power-number}

    1. Stop and remove the current container:
    ```sh
    docker stop pmm-server && docker rm pmm-server
    ```
    2. Pull the latest PMM 2 image:
    ```sh
    docker pull percona/pmm-server:2
    ```
    3. Run the updated container with your existing volume:
    ```sh
    docker run -d -v pmm-data:/srv -p 443:443 --name pmm-server --restart always percona/pmm-server:2
    ```
    4. Verify the version shows 2.44.1 in the PMM UI.


## Step 3: Migrate to the latest PMM 3

Once your server is running PMM 2.44.1, try migrating directly to the latest PMM 3 version using one of the methods below:
 

=== "Automated Docker migration (Recommended)"
    Use this upgrade script for a simplified migration process.
    { .power-number}
    
    1. Download and prepare the automated migration script:   
	```sh
	curl -o get-pmm.sh https://www.percona.com/get/pmm
	```
    2. Make the script executable: 
    ```sh
    chmod +x get-pmm.sh
    ```
    3. Run the migration script with the `-b` flag to create a backup of your PMM 2 instance before the migration:
    ```sh
    ./get-pmm.sh -n <container-name> -b
    ```
    4. Note the backup volume name displayed during the migration (e.g., `pmm-data-2025-01-16-165135`) so that you can restore this backup if needed.
    5. Check additional script options:
    ```sh
    ./get-pmm.sh -h
    ```
    !!! note alert alert-primary "Restore PMM 2 backup"
        If you need to revert to the PMM 2 instance, restore the backup created above:
        { .power-number}

        1. Stop the PMM 3 container:
            ```sh
            docker stop pmm-server
            ```
        2. Start a PMM 2 container using the backup volume, replacing `<backup-volume-name>` (e.g., `pmm-data-2025-01-16-165135`) with your actual backup volume name:

            ```sh
            docker run -d -p 443:443 --volume <backup-volume-name>:/srv --name pmm-server --restart always percona/pmm-server:2.44.0
            ```
        3. Verify that your PMM 2 instance is running correctly and all your data is accessible.

=== "Manual migration (Docker/Kubernetes/Podman/AMI/OVF)"
    === "Docker with volume"
        Follow these manual steps to migrate your PMM 2 Server to PMM 3:
        { .power-number}

        1. Stop all PMM Server services:

            ```sh
            docker exec -t <pmm-server> supervisorctl stop all
            ```

        2. Transfer `/srv` directory ownership:

            ```sh
            docker exec -t <pmm-server> chown -R pmm:pmm /srv
            ```

        3. List and note down your Docker volume:
       
            ```sh
            {% raw %}
            docker inspect -f '{{ range .Mounts }}{{ if eq .Type "volume" }}{{ .Name }}{{ "\n" }}{{ end }}{{ end }}' <pmm-server>
            {% endraw %}
            ```

        4. Stop and remove existing container:

            ```sh
            docker stop pmm-server && docker rm pmm-server
            ```

        5. Pull PMM 3 Server image:

            ```sh
            docker pull percona/pmm-server:3
            ```

        6. Run the new version of PMM Server with the existing volume:
       
            ```sh
            docker run -d -v pmm-server-data:/srv -p 443:8443 --name pmm-server --restart always percona/pmm-server:3
            ```

    === "Docker with data container"
        Follow these manual steps to upgrade your PMM 2 Server to PMM 3:
        { .power-number}

        1. Stop all PMM Server services:

            ```sh
            docker exec -t <pmm-server> supervisorctl stop all
            ```

        2. Transfer `/srv` directory ownership:

            ```sh
            docker exec -t <pmm-server> chown -R pmm:pmm /srv
            ```

        3. Identify the data container using either:
       
            ```sh
            docker ps -a --filter "status=created"
            ```
           
            OR

            ```sh
            {% raw %}
            docker inspect -f '{{ range .Mounts }}{{ if eq .Type "volume" }}{{ .Name }}{{ "\n" }}{{ end }}{{ end }}' <pmm-server>
            {% endraw %}
            ```
            
        4. Stop and remove the existing container:

            ```sh
            docker stop pmm-server && docker rm pmm-server
            ```

        5. Pull PMM 3 Server image:
       
            ```sh
            docker pull percona/pmm-server:3
            ``` 

        6. Run the new version of PMM Server with the existing data container:

            ```sh
            docker run -d --volumes-from pmm-server-data -p 443:8443 --name pmm-server --restart always percona/pmm-server:3
            ```

    === "Helm"
        Follow these steps to migrate your PMM 2 Server deployed with Helm to PMM 3:
        {.power-number}

        1. Update the Percona Helm repository:

            ```sh
            helm repo update percona
            ```

        2. Export current values to a file:

            ```sh
            helm show values percona/pmm > values.yaml
            ```

        3. Update the `values.yaml` file to match your PMM 2 configuration

        4. Stop all PMM Server services:

            ```sh
            kubectl exec pmm-0 -- supervisorctl stop all
            ```

        5. Transfer `/srv` directory ownership:

            ```sh
            kubectl exec pmm-0 -- chown -R pmm:pmm /srv
            ```

        6. Upgrade PMM using Helm:

            ```sh
            helm upgrade pmm -f values.yaml --set podSecurityContext.runAsGroup=null --set podSecurityContext.fsGroup=null percona/pmm
            ```

        7. If Kubernetes did not trigger the upgrade automatically, delete the pod to force recreation:
            ```sh
            kubectl delete pod pmm-0
            ```

    === "Podman"
        Follow these steps to migrate to PMM 3 a PMM 2 Server deployed with Podman:
        {.power-number}

        1. Pull the PMM 3 Server image:

            ```sh
            podman pull percona/pmm-server:3
            ```

        2. Stop all PMM Server services:

            ```sh
            podman exec pmm-server supervisorctl stop all
            ```

        3. Transfer `/srv` directory ownership:
            ```sh
            podman exec pmm-server chown -R pmm:pmm /srv
            ```

        4. Remove the existing systemd service file:

            ```sh
            rm ~/.config/systemd/user/pmm-server.service
            ```

        5. Follow the installation steps from the [PMM 3 Podman installation guide](../install-pmm/install-pmm-server/deployment-options/podman/index.md) to complete the upgrade.

    === "AMI/OVF instance"
        Follow these steps to migrate a PMM 2 Server deployed as an AMI/OVF instance to PMM 3:
        {.power-number}

        1. Back up your current instance and keep your PMM 2 instance running until confirm a successful migration.

        2. Deploy a new PMM 3 AMI/OVF instance.

        3. On the new instance, stop the Podman service:

            ```sh
            systemctl --user stop pmm-server
            ```

        4. Clear the service volume directory:

            ```sh
            rm -rf /home/admin/volume/srv/*
            ```

        5. On the old instance, stop all services:

            ```sh
            sudo supervisorctl stop all
            ```

        6. Transfer data from old to new instance:

            ```sh
            sudo scp -r /srv/* admin@newhost:/home/admin/volume/srv
            ```

        7. Set proper permissions on the new instance:

            ```sh
            chown -R admin:admin /home/admin/volume/srv/
            ```

        8. Start the PMM service on the new instance:

            ```sh
            systemctl --user start pmm-server
            ```

        9. Verify that PMM 3 is working correctly with the migrated data.
        
        10. Update PMM Client configurations by editing the `/usr/local/percona/pmm2/config/pmm-agent.yml` with the new server address, then restart the PMM Client.

        !!! note alert alert-primary "Revert AMI/OVF instance to PMM 2"
            If you need to restore to the PMM 2 instance after the migration:
            {.power-number}

            1. Access old instance via SSH.
            2. Start services: `supervisorctl start all`.
            3. Update client configurations to point to old instance.

If the migration succeeds, skip to [Step 5: Migrate PMM 2 Clients to PMM 3](#step-5-migrate-pmm-2-clients-to-pmm-3)

## Step 4: If direct migration fails, migrate through PMM 3.7.0

If Step 3 didn't work, restore your PMM 2 backup from Step 1, then repeat the same migration steps but use the PMM 3.7.0 image tag instead:

- Docker: `percona/pmm-server:3.7.0` instead of `percona/pmm-server:3`
- Helm: Use `--set image.tag=3.7.0` or pin the chart version with `--version` to deploy PMM 3.7.0
- Automated script: `./get-pmm.sh -n <container-name> -t 3.7.0 -b`

Once you're running PMM 3.7.0, upgrade to the latest version using the standard upgrade method for your deployment:

- [Upgrade PMM Server using Docker](upgrade_docker.md)
- [Upgrade PMM Server using Podman](upgrade_podman.md)
- [Upgrade PMM Server using Helm](upgrade_helm.md)

!!! caution alert alert-warning "Important"
    PMM 2 Clients are deprecated. Compatibility with PMM Server 3.4.0 and later is not guaranteed, and transitional support will be removed in a future release. Upgrade to PMM 3 Client as soon as possible to ensure full functionality.

## Step 5: Migrate PMM 2 Clients to PMM 3

PMM 3 Server provides limited support for PMM 2 Clients (metrics and Query Analytics only). Upgrade to PMM 3 Client as soon as possible to ensure full functionality.

How you upgrade depends on how your PMM Server was set up:

=== "Server migrated from PMM 2 to PMM 3"
    If you migrated your PMM 2 Server to PMM 3 (following Step 3 or Step 4), the server automatically removed legacy inventory prefixes during migration. You can [upgrade each PMM Client from v2 to v3 directly](../pmm-upgrade/upgrade_client.md) without unregistering first.

=== "PMM 2 Client added to a fresh PMM 3 Server"
    If you added a PMM 2 Client to a PMM 3 Server that was not migrated from v2, inventory prefixes are not removed automatically. Unregister the client before upgrading:
    {.power-number}

    1. Unregister the PMM 2 Client:
    ```sh
    pmm-admin unregister
    ```
    2. [Upgrade to PMM 3 Client](../pmm-upgrade/upgrade_client.md).
    3. [Configure the PMM 3 Client](../install-pmm/install-pmm-client/package_manager.md#step-2-install-pmm-client) to connect to your PMM Server using service accounts.

## Step 6: Migrate your API keys to service accounts

PMM 3 replaces API keys with service accounts to enhance security and simplify access management. You can trigger this API key conversion from the UI or from the CLI.

=== "From CLI"
    You can also initiate the conversion using the following command. 
    
    Be sure to replace `admin:admin` with your credentials and update the server address to match your PMM Server address (the same URL you use to access the PMM web interface):

    ```sh
    curl -X POST -k https://<pmm-server-address>/graph/api/serviceaccounts/migrate \
    -u admin:admin \
    -H "Content-Type: application/json"
    ```

    The response will display the migration details:
    !!! example "Expected output"
        ```
        {"total":3,"migrated":3,"failed":0,"failedApikeyIDs":[],"failedDetails":[]}
        ```    
 
=== "From the UI"
    PMM automatically migrates existing API keys to service accounts when you first log in as an Admin user. The migration results are displayed in a popup dialog box.
    If no popup appears, it likely means there are no API keys to migrate. This is typical for PMM Servers without connected services.

### Verify the conversion
	
To verify that API keys were successfully migrated, go to **Users and Access > Service accounts**, where you can check the list of service accounts available and confirm that the **API Keys** menu is no longer displayed.

If any API keys fail to migrate, you can either: 

- delete the problematic API keys and create new service accounts
- keep using the existing API keys until you're ready to replace them

### Post-migration steps

After you finish migrating PMM:
{.power-number}

1. Verify that all PMM Clients are up to date by checking **Configuration > Updates**.
2. Confirm all previously monitored services are reporting correctly to the new PMM 3 Server by reviewing **Inventory > Services**.
3. Check the dashboards to make sure you're receiving the metrics and QAN data.

### Variables for migrating from PMM v2 to PMM v3

When migrating from PMM v2 to PMM v3, you'll need to update your environment variables to match the new naming convention. This is because PMM v3 introduces several important changes to improve consistency and clarity:

- environment variables now use `PMM_` prefix
- some boolean flags reversed (e.g., `DISABLE_` → `ENABLE_`)
- removed deprecated variables

### Examples

```bash
# PMM v2
-e DISABLE_UPDATES=true -e DATA_RETENTION=720h

# PMM v3 equivalent
-e PMM_ENABLE_UPDATES=false -e PMM_DATA_RETENTION=720h
```

#### Migration reference table

The following table lists all the environment variable changes between PMM v2 and PMM v3. Review this table when updating your deployment configurations.

??? note "Click to expand migration reference table"

    #### Configuration variables
    | PMM 2                           | PMM 3                              | Comments                     |
    |---------------------------------|------------------------------------|------------------------------|
    | `DATA_RETENTION`                | `PMM_DATA_RETENTION`               |                              |
    | `DISABLE_ALERTING`              | `PMM_ENABLE_ALERTING`              |                              |
    | `DISABLE_UPDATES`               | `PMM_ENABLE_UPDATES`               |                              |
    | `DISABLE_TELEMETRY`             | `PMM_ENABLE_TELEMETRY`             |                              |
    | `DISABLE_BACKUP_MANAGEMENT`     | `PMM_ENABLE_BACKUP_MANAGEMENT`     | Note the reverted boolean    |
    | `ENABLE_AZUREDISCOVER`          | `PMM_ENABLE_AZURE_DISCOVER`        |                              |
    | `ENABLE_RBAC`                   | `PMM_ENABLE_ACCESS_CONTROL`        |                              |
    | `LESS_LOG_NOISE`                |                                    | Removed in PMM v3            |
    
    #### Metrics configuration
    | PMM 2                           | PMM 3                              | 
    |---------------------------------|------------------------------------|
    | `METRICS_RESOLUTION`            | `PMM_METRICS_RESOLUTION`           | 
    | `METRICS_RESOLUTION_HR`         | `PMM_METRICS_RESOLUTION_HR`        | 
    | `METRICS_RESOLUTION_LR`         | `PMM_METRICS_RESOLUTION_LR`        | 
    | `METRICS_RESOLUTION_MR`         | `PMM_METRICS_RESOLUTION_MR`        |    
    
    
    #### ClickHouse configuration
    | PMM 2                               | PMM 3                              | Comments                 |
    |-------------------------------------|------------------------------------|--------------------------|
    | `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`  | `PMM_CLICKHOUSE_ADDR`              |                          |
    | `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE` | `PMM_CLICKHOUSE_DATABASE`         |                        |
    | `PERCONA_TEST_PMM_CLICKHOUSE_DATASOURCE` | `PMM_CLICKHOUSE_DATASOURCE`      |                       |
    | `PERCONA_TEST_PMM_CLICKHOUSE_HOST`  | `PMM_CLICKHOUSE_HOST`              |                          |
    | `PERCONA_TEST_PMM_CLICKHOUSE_PORT`  | `PMM_CLICKHOUSE_PORT`              |                          |
    | `PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE` | `PMM_DISABLE_BUILTIN_CLICKHOUSE` |                  |
    | `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE` |                                    | Removed in PMM v3, new version|
    | `PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`  |                                    | Removed in PMM v3, new version|
    
    #### PostgreSQL configuration
    | PMM 2                               | PMM 3                              | 
    |-------------------------------------|------------------------------------|
    | `PERCONA_TEST_POSTGRES_ADDR`        | `PMM_POSTGRES_ADDR`                |
    | `PERCONA_TEST_POSTGRES_DBNAME`      | `PMM_POSTGRES_DBNAME`              |
    | `PERCONA_TEST_POSTGRES_USERNAME`    | `PMM_POSTGRES_USERNAME`            | 
    | `PERCONA_TEST_POSTGRES_DBPASSWORD`  | `PMM_POSTGRES_DBPASSWORD`          |  
    | `PERCONA_TEST_POSTGRES_SSL_CA_PATH` | `PMM_POSTGRES_SSL_CA_PATH`         | 
    | `PERCONA_TEST_POSTGRES_SSL_CERT_PATH` | `PMM_POSTGRES_SSL_CERT_PATH`      | 
    | `PERCONA_TEST_POSTGRES_SSL_KEY_PATH` | `PMM_POSTGRES_SSL_KEY_PATH`       |   
    | `PERCONA_TEST_POSTGRES_SSL_MODE`    | `PMM_POSTGRES_SSL_MODE`            |  
    | `PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES` | `PMM_DISABLE_BUILTIN_POSTGRES` |   
   
    #### Telemetry & development
    | PMM 2                               | PMM 3                              | 
    |-------------------------------------|------------------------------------|
    | `PMM_TEST_TELEMETRY_DISABLE_SEND`   | `PMM_DEV_TELEMETRY_DISABLE_SEND`   |                
    | `PERCONA_TEST_TELEMETRY_DISABLE_START_DELAY` | `PMM_DEV_TELEMETRY_DISABLE_START_DELAY` | 
    | `PMM_TEST_TELEMETRY_FILE`           | `PMM_DEV_TELEMETRY_FILE`           |   
    | `PERCONA_TEST_TELEMETRY_HOST`       | `PMM_DEV_TELEMETRY_HOST`           |   
    | `PERCONA_TEST_TELEMETRY_INTERVAL`   | `PMM_DEV_TELEMETRY_INTERVAL`       |      
    | `PERCONA_TEST_TELEMETRY_RETRY_BACKOFF` | `PMM_DEV_TELEMETRY_RETRY_BACKOFF` |   
    | `PERCONA_TEST_VERSION_SERVICE_URL`  | `PMM_DEV_VERSION_SERVICE_URL`      |         
    | `PERCONA_TEST_STARLARK_ALLOW_RECURSION` | `PMM_DEV_ADVISOR_STARLARK_ALLOW_RECURSION` |       
    
    #### Removed variables
    | PMM 2                               | PMM 3                              | Comments                     |
    |-------------------------------------|------------------------------------|------------------------------|
    | `PERCONA_TEST_AUTH_HOST`            |                                    | Removed, use `PMM_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_HOST`          |                                    | Removed, use `PMM_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_INTERVAL`      |                                    | Removed, not used            |
    | `PERCONA_TEST_CHECKS_PUBLIC_KEY`    |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY` |
    | `PERCONA_TEST_NICER_API`            |                                    | Removed in PMM v3            |
    | `PERCONA_TEST_SAAS_HOST`            |                                    | Removed, use `PMM_PERCONA_PLATFORM_ADDRESS` |

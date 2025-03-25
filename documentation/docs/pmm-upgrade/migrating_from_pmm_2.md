# Migrate PMM 2 to PMM 3

PMM 3 introduces significant architectural changes that require a gradual transition from PMM 2.

You can migrate to PMM 3 either automatically using the automated migration script (recommended) or manually by following step-by-step instructions.

To gradually migrate to PMM 3:

## Step 1: Upgrade PMM 2 Server to the latest version

Before migrating PMM 2 to PMM 3, ensure your PMM 2 Server is running the latest version:
{.power-number}

1. From the **Home** page, scroll to the **PMM Upgrade** panel and click the Refresh button to check for updates manually.
2. If an update is available, click the **Update** button to install the latest PMM 2 version.
3. Verify that the update was successful by checking the version number after the update completes.

## Step 2: Migrate PMM 2 Server to PMM 3

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
    3. Run the migration script with the `-b` flag to create a backup of your PMM2 instance before the migration:

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

## Step 3: Migrate PMM 2 Clients to PMM 3

!!! caution alert alert-warning "Important"
    PMM 3 Server provides limited support for PMM 2 Clients (metrics and Query Analytics only). This support will be removed in PMM 3.3.

Depending on your initial installation method, update PMM Clients using your operating system's package manager or using a tarball.
For detailed instructions, see the [Upgrade PMM Client topic](../pmm-upgrade/upgrade_client.md).

## Step 4: Migrate your API keys to service accounts

PMM 3 replaces API keys with service accounts to enhance security and simplify access management. You can trigger this API key conversion from the UI or from the CLI.

=== "From CLI"
    You can also initiate the conversion using the following command. 
    Be sure to replace `admin:admin` with your credentials and update the server address (`localhost` or `127.0.0.1`) and port number (`3000`) if they differ from the defaults:
    
    ```sh
    curl -X POST http://localhost:3000/api/serviceaccounts/migrate \
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
    If no popup appears, it likely means there are no API keys to migrate—this is typical for PMM Servers without connected services.

### Verify the conversion
	
To verify that API keys were successfully migrated, go to **Administration > Users and Access > Service Accounts**, where you can check the list of service accounts available and confirm that the **API Keys** menu is no longer displayed.

If any API keys fail to migrate, you can either: 

- delete the problematic API keys and create new service accounts
- keep using the existing API keys until you're ready to replace them

### Post-migration steps

After you finish migrating PMM:
{.power-number}

1. Verify that all PMM Clients are up to date by checking **PMM Configuration > Updates**.
2. Confirm all previously monitored services are reporting correctly to the new PMM 3 Server by reviewing **Configuration > PMM Inventory > Services**.
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
    | PMM 2                          | PMM 3                              | Comments                      |
    |---------------------------------|------------------------------------|------------------------------|
    | `DATA_RETENTION`                | `PMM_DATA_RETENTION`               |                              |
    | `DISABLE_ALERTING`              | `PMM_ENABLE_ALERTING`              |                              |
    | `DISABLE_UPDATES`               | `PMM_ENABLE_UPDATES`               |                              |
    | `DISABLE_TELEMETRY`             | `PMM_ENABLE_TELEMETRY`             |                              |
    | `DISABLE_BACKUP_MANAGEMENT`      | `PMM_ENABLE_BACKUP_MANAGEMENT`     | Note the reverted boolean   |
    | `ENABLE_AZUREDISCOVER`          | `PMM_ENABLE_AZURE_DISCOVER`        |                              |
    | `ENABLE_RBAC`                   | `PMM_ENABLE_ACCESS_CONTROL`        |                              |
    | `LESS_LOG_NOISE`                |                                    | Removed in PMM v3            |
    
    #### Metrics configuration
    | PMM 2                          | PMM 3                              | 
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
    | `PERCONA_TEST_AUTH_HOST`            |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_HOST`          |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_INTERVAL`      |                                    | Removed, not used            |
    | `PERCONA_TEST_CHECKS_PUBLIC_KEY`    |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY` |
    | `PERCONA_TEST_NICER_API`            |                                    | Removed in PMM v3            |
    | `PERCONA_TEST_SAAS_HOST`            |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |

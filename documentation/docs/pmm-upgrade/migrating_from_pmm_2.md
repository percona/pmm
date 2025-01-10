# Migrate PMM 2 to PMM 3

PMM 3 introduces significant architectural changes that require gradual transition from PMM 2.

You can migrate to PMM 3 either automatically using the upgrade script (recommended), or manually by following step-by-step instructions.

To graduallly migrate to PMM 3:

## Step 1: Upgrade PMM 2 Server to the latest version

Before upgrading to PMM 3, ensure your PMM 2 Server is running the latest version:
{.power-number}

1. From the **Home** page, scroll to the **PMM Upgrade** panel and click the Refresh button to manually check for updates.
2. If an update is available, click the **Update** button to install the latest PMM 2 version.
3. Verify the update was successful by checking the version number after the update completes.

## Step 2: Migrate PMM 2 Server to PMM 3

=== "Automated upgrade (Recommended)"
    Use this upgrade script for a simplified migration process:

    1. Download and run the following script to start the upgrade. The `-b` flag ensures your data is backed up before the upgrade.

        ```sh
        ./get-pmm.sh -n <container-name> -b
        ```

    2. Check additional script options:
        ```sh
        ./get-pmm.sh -h
        ```

=== "Manual upgrade from PMM 2 with Docker volume"

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

    6. Run new new version of PMM Server with the existing volume:
   
        ```sh
        docker run -d -v pmm-server-data:/srv -p 443:8443 --name pmm-server --restart always percona/pmm-server:3
        ```

=== "Manual upgrade from PMM 2 with data container"

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

## Step 3: Migrate PMM 2 Clients to PMM 3

!!! caution alert alert-warning "Important"
    PMM 3 Server provides limited support for PMM 2 Clients (metrics and Query Analytics only). This support will be removed in PMM 3.3.

Depending on your initial installation method, update PMM Clients using your operating system's package manager or using a tarball.
For detailed instructions, see the [Upgrade PMM Client topic](../pmm-upgrade/upgrade_client.md).

### Post-migration steps

After you finish migrating:
{.power-number}

1. Verify that all PMM Clients are up to date by checking **PMM Configuration > Updates**.
2. Confirm all previously monitored services are reporting correctly to the new PMM 3 Server by reviewing **Configuration > PMM Inventory > Services**.
3. Check the dashboards to make sure you're receiving the metrics and QAN data.

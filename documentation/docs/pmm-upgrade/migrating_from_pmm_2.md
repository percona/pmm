# Migrate PMM 2 to PMM 3

PMM 3 introduces significant architectural changes that require gradual transition from PMM 2.

You can migrate to PMM 3 either automatically using the automated migration script (recommended), or manually by following step-by-step instructions.

To graduallly migrate to PMM 3:

## Step 1: Upgrade PMM 2 Server to the latest version

Before migrating PMM 2 to PMM 3, ensure your PMM 2 Server is running the latest version:
{.power-number}

1. From the **Home** page, scroll to the **PMM Upgrade** panel and click the Refresh button to manually check for updates.
2. If an update is available, click the **Update** button to install the latest PMM 2 version.
3. Verify the update was successful by checking the version number after the update completes.

## Step 2: Migrate PMM 2 Server to PMM 3

=== "Automated Docker migration (Recommended)"
    Use this upgrade script for a simplified migration process:
    { .power-number}

    1. Download and run the [automated migration script](https://www.percona.com/get/pmm) to start the migration. The `-b` flag creates a backup of your PMM2 instance to ensure that your data is backed up before the migration.

        ```sh
        ./get-pmm.sh -n <container-name> -b
        ```
    2. Note the backup volume name displayed during the migration (e.g., `pmm-data-2025-01-16-165135`) so that you can restore this backup if needed.

    3. Check additional script options:
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

        6. Run new new version of PMM Server with the existing volume:
       
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

        5. Follow the installation steps from the [PMM 3 Podman installation guide](../install-pmm/install-pmm-server/baremetal/podman/index.md) to complete the upgrade.

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
	
### From the UI
PMM automatically migrates existing API keys to service accounts when you first log in as an Admin user. The migration results are displayed in a popup dialog box. 

If no popup appears, it likely means there are no API keys to migrate—this is typical for PMM Servers without connected services.
	
### From CLI
You can also initiate the conversion using the following command. 
Be sure to replace `admin:admin` with your credentials and update the server address (`localhost` or `127.0.0.1`) and port number (`3000`) if they differ from the defaults:

	
```sh
curl -X POST http://localhost:3000/api/serviceaccounts/migrate \
-u admin:admin \
-H "Content-Type: application/json
```
	
The response will display the migration details:

!!! example "Expected output"

	```
	{"total":3,"migrated":3,"failed":0,"failedApikeyIDs":[],"failedDetails":[]}
	```    
	
### Verify the conversion
	
To verify the that API keys were successfully migrated, go to **Administration > Users and Access > Service Accounts** where you can check the list of service accounts available and confirm that the **API Keys** menu is no longer displayed.

If any API keys fail to migrate, you can either: 

- delete the problematic API keys and create new service accounts
- keep using the existing API keys until you're ready to replace them

### Post-migration steps

After you finish migrating PMM:
{.power-number}

1. Verify that all PMM Clients are up to date by checking **PMM Configuration > Updates**.
2. Confirm all previously monitored services are reporting correctly to the new PMM 3 Server by reviewing **Configuration > PMM Inventory > Services**.
3. Check the dashboards to make sure you're receiving the metrics and QAN data.

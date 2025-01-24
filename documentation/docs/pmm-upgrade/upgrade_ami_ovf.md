# Manual upgrade: Upgrade PMM Server using AMI/OVF

## Before you begin
Before starting the upgrade, complete these preparation steps:
{.power-number}

1. Create a backup before upgrading, as downgrades are not possible. 
**What should be the link to creatinh AMI backup?**
2. Do not shut down your old PMM 2 instance until you verify that the migration to PMM v3 was successful.

## Upgrade steps

Follow these steps to upgrade your PMM Server while preserving your monitoring data and settings:
{.power-number}

1. Deploy a new PMM 3 AMI/OVF instance.

2. Access the new instance via SSH and stop the Podman service:
    ```sh
    systemctl --user stop pmm-server
    ```

3. Clear the service volume directory:
    ```sh
    rm -rf /home/admin/volume/srv/*
    ```

4. Access the old instance via SSH and stop all services (run as `root`):
    ```sh
    supervisorctl stop all
    ```

5. Transfer data from the old instance to the new one (run as `root`):
    ```sh
    scp -r /srv/* admin@newhost:/home/admin/volume/srv
    ```

6. Set proper ownership on the new instance:
    ```sh
    chown -R admin:admin /home/admin/volume/srv/
    ```

7. Start the PMM service on the new instance:
    ```sh
    systemctl --user start pmm-server
    ```

8. Verify that PMM 3 works and contains the migrated data.

9. Update PMM Client configurations by editing `/usr/local/percona/pmm2/config/pmm-agent.yml` with the new Server address and restart the PMM Client.

10. [Migrate PMM 2 Clients to PMM 3](../pmm-upgrade/migrating_from_pmm_2.md#step-3-migrate-pmm-2-clients-to-pmm-3).

## Restore PMM 2 instance from backup 

**is backup a good word in the title?**

If you need to restore the old PMM 2 instance:
{.power-number}

1. Access the old instance via SSH.
2. Start all services:
    ```sh
    supervisorctl start all
    ```
3. Update PMM Client configurations to point back to the old instance.
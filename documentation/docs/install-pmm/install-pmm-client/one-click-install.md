# Quick-Install PMM Client 


!!! warning "Technical Preview status"
    This feature is not production-ready yet. Try it out and share your feedback on the [PMM forum](https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-3/84) or [JIRA Tracker](https://perconadev.atlassian.net/jira/software/c/projects/PMM/issues/).

Instead of manually installing PMM Client, registering the node, and adding a database service separately, you can do it all with a single command generated from the PMM UI.

## Prerequisites

- A running PMM Server instance.
- A Linux host with root or sudo access where your database is running.
- A database monitoring user with the appropriate permissions for your database type.
- The PMM Server must be reachable from the database host over HTTPS.

## Generate and run the install command

Generate and run the install command from the PMM UI:
{.power-number}

1. Log in to the PMM web interface as an Admin user.

2. In the left sidebar, go to **Inventory > Install PMM Client**.

3. Select your database technology from the **Technology** dropdown: **MySQL**, **PostgreSQL**, **MongoDB**, or **Valkey**.

4. Check the **PMM host** field. It is pre-filled with the current server's hostname. Change it only if your database host connects to PMM Server at a different address. Do not include http/https or paths.

5. Click **Generate install token**. PMM creates a temporary Admin-level token and fills it into the **Service token** field automatically. The token expires in 15 minutes. Do not share the generated command as it contains an Admin-level token.

6. By default, the command prompts for your database credentials when you run it on the terminal. To skip the prompt, toggle on **Running in CI/automation?** and enter your database username and password before copying the command.

7. (Optional) Expand **Advanced options** to customize your service name, port, and node settings.

8. Copy the generated command and run it on your database host with `sudo`:

    ```sh
    sudo -E bash '/tmp/install-pmm-client.sh' \
        --pmm-server-url 'https://service_token:<token>@<pmm-server>' \
        --tech '<technology>'
    ```

    The command installs PMM Client, registers the node with PMM Server, and adds the first monitored service automatically.

9. Verify that the node and service appear in PMM under **Inventory > Services**.

## Add more databases on the same node

To monitor a second database on the same node, generate a new command for the additional technology and set a different port in **Advanced options**. The command skips node re-registration after the first run, so your existing monitored services are not affected.

## Re-add a previously removed service

If you removed a service from PMM and the command now fails with an authentication error, the token has expired. Regenerate the command here and re-run it. The command re-adds the service without affecting other services on the node.

!!! warning
    Do not use **Force re-register node** in **Advanced options** when re-adding a service. This removes the node and all its services from PMM and should only be used to recover from a failed first install.

## Related topics

- [Connect database services](connect-database/index.md)
- [PMM Client installation overview](index.md)
- [Install PMM Client with Package Manager](package_manager.md)

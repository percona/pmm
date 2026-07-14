# Install PMM Client with one command (Dev Preview)

!!! warning "Dev Preview"
    This feature is in Dev Preview. It is not recommended for production use.

The **Install PMM Client** page in the PMM UI generates a ready-to-run command that installs and configures `pmm-client` on your database host in a single step. You no longer need to manually install packages, create service accounts, or run `pmm-admin config` and `pmm-admin add` separately.

The same generated command is also available from **Inventory → Add Node** in Grafana.

## Prerequisites

- A running PMM Server instance.
- A Linux host with root or sudo access where your database is running.
- A database monitoring user with the appropriate permissions for your database type.
- The PMM Server must be reachable from the database host over HTTPS.

## Generate and run the install command

{.power-number}

1. Log in to the PMM web interface as an Admin user.

2. In the left sidebar, click **Install PMM Client**.

3. Select your database technology: **MySQL**, **PostgreSQL**, **MongoDB**, or **Valkey**.

4. Click **Generate short-lived token**. PMM creates a temporary service-account token and fills it into the command automatically. The token is embedded in the PMM Server URL as `https://service_token:<token>@<host>` and is only visible in your browser — it is never sent to or stored by Percona.

5. (Optional) Expand **Advanced options** to set a custom node name, port, or service name.

6. Copy the generated command and run it on the target database host:

    ```sh
    sudo -E bash '/tmp/install-pmm-client.sh' \
        --pmm-server-url 'https://service_token:<token>@<pmm-server>' \
        --tech '<technology>'
    ```

    The script:
    
    - Installs `pmm-client` using the host's package manager.
    - Starts `pmm-agent` (via systemd, or `nohup` as a fallback in environments without systemd).
    - Registers the node with PMM Server using the embedded token — no manual Grafana login required on the host.
    - Adds the first monitored service with `pmm-admin add`.

7. When prompted, enter your database username and password. If you prefer not to enter credentials interactively, set them as environment variables before running the command:

    === "MySQL"
        ```sh
        export DB_USER=<user>
        export DB_PASSWORD=<password>
        ```

    === "PostgreSQL"
        ```sh
        export DB_USER=<user>
        export DB_PASSWORD=<password>
        ```

    === "MongoDB"
        ```sh
        export DB_USER=<user>
        export DB_PASSWORD=<password>
        ```

    === "Valkey"
        ```sh
        export DB_PASSWORD=<password>
        ```

    !!! note
        The generated command never embeds database credentials. Environment variables take precedence over interactive prompts, making this approach suitable for automation.

8. Verify that the node and service appear in PMM under **Inventory → Services**.

## Limitations

- **One service per run**: Each run of the script adds a single monitored service. To monitor multiple databases on the same node, run the command separately for each database type.
- **Re-adding a service**: If you remove a service from PMM and then re-run the script for the same node, you will see a prompt to use `--force`. Using `--force` re-registers the node and removes all existing services on it. To re-add a single service without affecting others, use `pmm-admin add` directly instead.
- **Non-interactive environments**: If the script detects that stdin is not a TTY and no credentials are provided, it exits early with an error message. This prevents a partially registered node. Pass credentials as environment variables for non-interactive use.

## Related topics

- [Connect database services](connect-database/index.md)
- [PMM Client command reference](../../use/commands/pmm-admin/pmm-admin.md)
- [Install PMM Client with Package Manager](package_manager.md)

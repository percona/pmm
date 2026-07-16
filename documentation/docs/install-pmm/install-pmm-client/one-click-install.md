# Install PMM Client (One-step) — Dev Preview

!!! warning "Dev Preview"
    This feature is in Dev Preview. It is not recommended for production use.

The **Install PMM Client (One-step)** page in the PMM UI generates a ready-to-run command that installs and configures `pmm-client` on your database host in a single step. You no longer need to manually install packages, create service accounts, or run `pmm-admin config` and `pmm-admin add` separately.

## Prerequisites

- A running PMM Server instance.
- A Linux host with root or sudo access where your database is running.
- A database monitoring user with the appropriate permissions for your database type.
- The PMM Server must be reachable from the database host over HTTPS.

## Generate and run the install command

{.power-number}

1. Log in to the PMM web interface as an Admin user.

2. In the left sidebar, under **Inventory**, click **Install PMM Client**.

3. Select your database technology from the **Technology** dropdown: **MySQL**, **PostgreSQL**, **MongoDB**, or **Valkey**.

4. Check the **PMM host** field. It is pre-filled with the current server's hostname. Leave it as-is unless your database host needs to reach PMM Server at a different address. Do not include the protocol (`http`/`https`), paths, or query parameters.

5. Click **Generate short-lived token**. PMM creates a temporary service-account token and fills it into the **Service token** field automatically. The token is embedded in the generated command as `https://service_token:<token>@<host>` and is only visible in your browser.

6. By default, the script prompts for database credentials when you run it on the terminal. To skip the prompt, toggle on **Running in CI/automation?** and enter your database username and password. The credentials will be included in the generated command.

7. (Optional) Expand **Advanced options** to set a custom service name, node address, or port.

8. Copy the generated command and run it on the target database host:

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

9. Verify that the node and service appear in PMM under **Inventory → Services**.

## Limitations

- **One service per run**: Each run of the script adds a single monitored service. To monitor multiple databases on the same node, run the command separately for each database type.
- **Re-adding a service**: If you remove a service from PMM and then re-run the script for the same node, you will see a prompt to use `--force`. Using `--force` re-registers the node and removes all existing services on it. To re-add a single service without affecting others, use `pmm-admin add` directly instead.
- **Non-interactive environments**: If the script detects that stdin is not a TTY and no credentials are provided, it exits early with an error message. This prevents a partially registered node. Pass credentials as environment variables for non-interactive use.

## Related topics

- [Connect database services](connect-database/index.md)
- [PMM Client command reference](../../use/commands/pmm-admin/pmm-admin.md)
- [Install PMM Client with Package Manager](package_manager.md)

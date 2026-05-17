# Connect ClickHouse databases to PMM

ClickHouse is a high-performance, column-oriented database for online analytical processing (OLAP). Connect your ClickHouse instances to PMM to track server health, query performance, and replication status.

PMM monitors ClickHouse through one of two metrics sources:

- **Native** — ClickHouse exposes its own Prometheus endpoint when the server's `<prometheus>` section is enabled. PMM scrapes it directly.
- **Exporter** — for servers without the native endpoint (including ClickHouse older than 22.6), PMM runs the bundled `clickhouse_exporter`, which connects over the native protocol and emits the same metric families.

By default (`--metrics-source=auto`), PMM probes the native endpoint and falls back to the exporter automatically.

## Prerequisites

Before connecting ClickHouse to PMM, review the prerequisites for your monitoring setup:

=== ":material-server: Local ClickHouse monitoring"
    - [PMM Server is installed](../../install-pmm-server/index.md) and running.
    - [PMM Client is installed](../../install-pmm-client/index.md) and the nodes are registered with PMM Server.
    - Access to the ClickHouse instance (localhost or network accessible) over the native TCP port (default: `9000`).

=== ":material-cloud: Remote ClickHouse monitoring"
    - [PMM Server is installed](../../install-pmm-server/index.md) and running.
    - PMM Client has direct network access to the ClickHouse instance.
    - You have ClickHouse authentication credentials.

## Security setup

Create a dedicated, read-only ClickHouse user for monitoring instead of reusing an administrative account:

```sql
CREATE USER pmm IDENTIFIED BY 'StrongPassword123!';
GRANT SELECT ON system.* TO pmm;
```

The exporter and the Query Analytics agent only read from the `system` database.

## Add service to PMM

You can add your ClickHouse service to PMM either through the user interface or via the command line.

=== ":material-console: Via command line"

    === "Basic setup (auto-probe)"

        Add a local ClickHouse instance with default settings. PMM probes the native endpoint and falls back to the exporter:
        ```sh
        pmm-admin add clickhouse \
          clickhouse-primary \
          127.0.0.1:9000 \
          --username=pmm \
          --password=StrongPassword123!
        ```

    === "Force the exporter"

        Use the bundled `clickhouse_exporter` regardless of the native endpoint:
        ```sh
        pmm-admin add clickhouse \
          clickhouse-primary \
          127.0.0.1:9000 \
          --username=pmm \
          --password=StrongPassword123! \
          --metrics-source=exporter
        ```

    === "Force the native endpoint"

        Scrape the ClickHouse native Prometheus endpoint directly (requires the server `<prometheus>` section to be enabled):
        ```sh
        pmm-admin add clickhouse \
          clickhouse-primary \
          127.0.0.1:9000 \
          --username=pmm \
          --password=StrongPassword123! \
          --metrics-source=native \
          --native-metrics-port=9363
        ```

    === "With Query Analytics"

        Enable the ClickHouse query log agent (QAN). The server must have `log_queries=1`:
        ```sh
        pmm-admin add clickhouse \
          clickhouse-primary \
          127.0.0.1:9000 \
          --username=pmm \
          --password=StrongPassword123! \
          --qan
        ```

    === "With TLS connection"

        Add an instance with TLS security:
        ```sh
        pmm-admin add clickhouse \
          clickhouse-primary \
          clickhouse-server.example.com:9000 \
          --username=pmm \
          --password=StrongPassword123! \
          --tls \
          --tls-ca=/path/to/ca.pem \
          --tls-cert=/path/to/client-cert.pem \
          --tls-key=/path/to/client-key.pem
        ```

    Useful flags:

    | Flag | Purpose |
    |---|---|
    | `--metrics-source` | `auto` (default), `native`, or `exporter`. |
    | `--native-metrics-port` | Port of the native Prometheus endpoint (default: `9363`). |
    | `--qan` | Enable the Query Analytics query log agent. |
    | `--environment`, `--cluster`, `--replication-set` | Topology labels. |
    | `--custom-labels` | Comma-separated `key=value` labels. |
    | `--skip-connection-check` | Add the service without validating connectivity. |

=== ":material-web: Via UI"

    To add the service from the user interface:
    {.power-number}

    1. Go to **PMM Configuration > PMM Inventory > Services > Add Service**.
    2. Select **ClickHouse** service type.
    3. Fill in the **Main details** section:
        - **Service name**: e.g., `clickhouse-primary`. Defaults to `<hostname>-clickhouse`.
        - **Nodes**: Select the PMM node where the agent is running.
        - **Hostname/Port**: The address and native port (default: `9000`) of your instance.
        - **Username/Password**: ClickHouse authentication credentials.
    4. Configure **Labels** (optional): environment, cluster, replication set, region, and custom labels.
    5. Configure **Additional options**: skip connection check, TLS, and metrics source.
    6. Click **Add service** to complete the setup.

#### Confirmation message

If the service is added successfully, PMM displays:
```sh
ClickHouse Service added.
Service ID  : /service_id/abcd1234-5678-efgh-ijkl-mnopqrstuvwx
Service name: clickhouse-primary
```

## Verify your ClickHouse service

After adding your ClickHouse service to PMM, verify that it's properly connected and collecting data.

### Check service status

=== ":material-console: Via command line"

    {.power-number}

    1. List all ClickHouse services and their status:
        ```bash
        pmm-admin inventory list services --service-type=clickhouse
        ```
    2. List the ClickHouse exporter agents (when the exporter source is used):
        ```bash
        pmm-admin inventory list agents --agent-type=clickhouse_exporter
        ```
    3. Check the overall PMM Client status:
        ```bash
        pmm-admin status
        ```

=== ":material-web: Via UI"
    {.power-number}

    1. Navigate to **PMM Configuration > PMM Inventory**.
    2. In the **Services** tab, find your newly added ClickHouse service.
    3. Verify the **Service Name** and **Address** match your configuration.
    4. Check the **Status** column shows as *Active*.

### Verify data collection

{.power-number}

1. Open the **Home** dashboard and verify your ClickHouse service appears in the **Monitored DB Services** panel.
2. Navigate to the **ClickHouse** dashboards from the left menu.
3. Select your ClickHouse service from the drop-down menu and confirm that recent metrics appear.

## Remove a ClickHouse service

=== ":material-console: Via command line"

    Remove a ClickHouse service (this cascades to the exporter and QAN agent):
    ```bash
    pmm-admin remove clickhouse clickhouse-primary
    ```

=== ":material-web: Via UI"
    {.power-number}

    1. Navigate to **PMM Configuration > PMM Inventory**.
    2. Find your ClickHouse service in the **Services** tab.
    3. Click the **Remove** button in the **Options** column.
    4. Confirm the removal when prompted.

## Next steps

- [Configure alerts](../../../alert/index.md) for critical ClickHouse metrics.
- Use [PMM Inventory](../../../use/dashboard-inventory.md) to view and manage all monitored instances.
- [ClickHouse official documentation](https://clickhouse.com/docs)
- [pmm-admin command reference](../../../use/commands/pmm-admin/pmm-admin.md)

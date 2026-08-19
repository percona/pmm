# Use external ClickHouse with PMM

Use an external ClickHouse instance when you want PMM to store Query Analytics data outside the `pmm-server` container, on a host or cluster you manage yourself.

This applies to standalone PMM deployments. If you're running PMM HA Cluster, PMM's Helm chart deploys and manages ClickHouse for you automatically, using the [PMM HA environment variables](../../install-pmm/install-HA-clustered.md#pre-configured-ha-variables) instead of the ones on this page.

## Environment variables for external ClickHouse

PMM predefines certain flags that enable you to use ClickHouse parameters as environment variables.

To use ClickHouse as an external database instance, provide the following environment variables:

| Variable | Value | Description |
|----------|-------|--------------|
| `PMM_CLICKHOUSE_ADDR` | `hostname:port` | Host and port of the external ClickHouse database instance. |
| `PMM_CLICKHOUSE_HOST` | `hostname` | Hostname of the external ClickHouse database. |
| `PMM_CLICKHOUSE_PORT` | `port` | Port of the external ClickHouse database. |
| `PMM_CLICKHOUSE_USER` | `username` | Username to connect to the external ClickHouse database. |
| `PMM_CLICKHOUSE_PASSWORD` | `password` | User password to connect to the external ClickHouse database. |
| `PMM_CLICKHOUSE_DATASOURCE_USER` | `username` | Username for the [read-only Grafana data source user](#restrict-the-clickhouse-data-source-to-a-read-only-user), required in PMM 3.9.1 and later. |
| `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | `password` | Password for the [read-only Grafana data source user](#restrict-the-clickhouse-data-source-to-a-read-only-user), required in PMM 3.9.1 and later. |
| `PMM_DISABLE_BUILTIN_CLICKHOUSE` | `1` | Disables the built-in ClickHouse database instance. |

### Optional environment variables

| Variable | Value | Description |
|----------|-------|--------------|
| `PMM_CLICKHOUSE_DATABASE` | `database name` | Database name of the external ClickHouse database instance. |
 
#### Example

To use ClickHouse as an external database instance, run PMM in docker or podman with the specified variables for external ClickHouse:

```sh
-e PMM_CLICKHOUSE_ADDR=<hostname>:<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_CLICKHOUSE_DATASOURCE_USER=<datasource-username>
-e PMM_CLICKHOUSE_DATASOURCE_PASSWORD=<datasource-password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

Alternatively, you can use the `PMM_CLICKHOUSE_HOST` and `PMM_CLICKHOUSE_PORT` variables instead of `PMM_CLICKHOUSE_ADDR`.

```sh
-e PMM_CLICKHOUSE_HOST=<hostname>
-e PMM_CLICKHOUSE_PORT=<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_CLICKHOUSE_DATASOURCE_USER=<datasource-username>
-e PMM_CLICKHOUSE_DATASOURCE_PASSWORD=<datasource-password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

## Restrict the ClickHouse data source to a read-only user

As of PMM 3.9.1, the Grafana ClickHouse data source connects as a dedicated, read-only user instead of the privileged account PMM's Query Analytics service uses to write data into ClickHouse. 

This limits what a signed-in Grafana user can do through the data source, even one with the lowest-privilege [Viewer role](../../admin/roles/index.md).

Before upgrading to PMM 3.9.1, check whether you need to create this user yourself:

=== "Built-in ClickHouse"

    PMM creates and provisions this user for you automatically. No action is required.

=== "External ClickHouse"
 
    Before upgrading to PMM 3.9.1, complete the steps to keep the data source working:
    {.power-number}

    1. In your ClickHouse server configuration (`config.xml` or a drop-in file under `config.d/`), confirm that both settings are enabled:

        ```xml
        <access_control_improvements>
            <settings_constraints_replace_previous>true</settings_constraints_replace_previous>
            <select_from_system_db_requires_grant>true</select_from_system_db_requires_grant>
        </access_control_improvements>
        ```

        Both are enabled by default from ClickHouse 25.3. You only need to make changes if you are running an older version or have overridden these defaults.

        | Setting | Purpose |
        |---|---|
        | `settings_constraints_replace_previous` | Required for `CHANGEABLE_IN_READONLY` to work in step 2. Without it, `CREATE USER` fails with `NOT_IMPLEMENTED`. |
        | `select_from_system_db_requires_grant` | Prevents the read-only user from reading `system.query_log`, which contains the SQL text of every query run on the instance. |

        Restart ClickHouse after changing these settings. `SYSTEM RELOAD CONFIG` and `SYSTEM RELOAD USERS` do not reload `access_control_improvements`.

    2. Run the following on your external ClickHouse instance to create a dedicated read-only user for PMM. Make sure to replace `your-password` with a [strong, randomly generated password](#enhance-clickhouse-security-for-pmm) and its SHA256 hash:

        ```sql
        CREATE USER grafana IDENTIFIED WITH sha256_password BY 'your-password'
        SETTINGS readonly = 1, max_execution_time CHANGEABLE_IN_READONLY;
 
        GRANT SELECT ON pmm.* TO grafana;
        ```

        If you set `PMM_CLICKHOUSE_DATABASE` to something other than `pmm`, grant `SELECT` on that database instead. Grant nothing beyond `SELECT`. If you restrict the user by network, allow the PMM Server host, since Grafana connects from there.
 
        If you manage ClickHouse users through configuration files instead, use the XML equivalent:
 
        ```xml
        <clickhouse>
            <users>
                <grafana>
                    <password_sha256_hex>your-password-hash</password_sha256_hex>
                    <networks>
                        <ip>::/0</ip>
                    </networks>
                    <profile>readonly</profile>
                </grafana>
            </users>
            <profiles>
                <readonly>
                    <readonly>1</readonly>
                    <constraints>
                        <max_execution_time>
                            <changeable_in_readonly>1</changeable_in_readonly>
                        </max_execution_time>
                    </constraints>
                </readonly>
            </profiles>
        </clickhouse>
        ```

        `readonly = 1` prevents writes, DDL, and all setting changes. The only exception is `max_execution_time`: the Grafana ClickHouse plugin sets it on every query to enforce its query timeout. 
        
        Without `CHANGEABLE_IN_READONLY`, every query fails with *Cannot modify 'max_execution_time' setting in readonly mode`*. 
        
        Do not use `readonly = 2`. The [Grafana plugin documentation](https://grafana.com/docs/plugins/grafana-clickhouse-datasource/latest/configure/#clickhouse-user-and-permissions) advises against it.

    3. Set `PMM_CLICKHOUSE_DATASOURCE_USER` and `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` to this user's credentials, as shown in the [previous example](#example). 
    
    This step is required. If you do not configure the new credentials, PMM cannot authenticate to ClickHouse after the upgrade.

    PMM will not fall back to the privileged `PMM_CLICKHOUSE_USER` account. This prevents the fallback from reopening the vulnerability that this fix addresses.
    
## Enhance ClickHouse security for PMM

When configuring PMM to use an external ClickHouse instance, make sure to enforce robust security practices to protect sensitive data and prevent unauthorized access:

- enable SSL/TLS encryption for all connections
- ensure that your ClickHouse instance is properly secured and monitored
- disable empty passwords and plain text passwords
- define all ClickHouse users explicitly, including permissions, to prevent automatic creation of unsecured users without passwords.
- generate strong, random passwords for the dedicated PMM ClickHouse user. Use the following commands to generate a password and its SHA256 hash (useful for advanced ClickHouse configurations):

    ```sh
    PASSWORD=$(base64 < /dev/urandom | head -c12)
    echo "$PASSWORD" # note it down
    echo -n "$PASSWORD" | sha256sum | tr -d '-'
    ```

For more details, see the [ClickHouse user and roles settings](https://clickhouse.com/docs/operations/settings/settings-users).

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/guides/troubleshooting).
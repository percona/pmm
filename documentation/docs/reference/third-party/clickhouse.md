# Use external ClickHouse with PMM

You can use an external ClickHouse database instance outside the PMM Server container running on other hosts.

## Environment variables

PMM predefines certain flags that allow you to use ClickHouse parameters as environment variables.


To use ClickHouse as an external database instance, provide the following environment variables: 
 
`PMM_CLICKHOUSE_ADDR` -> hostname:port
:   Name of the host and port of the external ClickHouse database instance. 

`PMM_CLICKHOUSE_HOST` -> hostname
:   Hostname of the external ClickHouse database.

`PMM_CLICKHOUSE_PORT` -> port
:   Port of the external ClickHouse database.

`PMM_CLICKHOUSE_USER` -> username
:   Username to connect to the external ClickHouse database.

`PMM_CLICKHOUSE_PASSWORD` -> password
:   User password to connect to the external ClickHouse database.

`PMM_CLICKHOUSE_DATASOURCE_USER` -> username
:   Username the Grafana ClickHouse data source uses to read query analytics data. See [Create the read-only data source user](#create-the-read-only-data-source-user).

`PMM_CLICKHOUSE_DATASOURCE_PASSWORD` -> password
:   Password of the read-only data source user.

`PMM_DISABLE_BUILTIN_CLICKHOUSE` -> 1
:   Disables the built-in ClickHouse database instance.

**Optional environment variables**

`PMM_CLICKHOUSE_DATABASE` -> database name
:   Database name of the external ClickHouse database instance.
 
**Example**

To use ClickHouse as an external database instance, run PMM in docker or podman with the specified variables for external ClickHouse:

```sh
-e PMM_CLICKHOUSE_ADDR=<hostname>:<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_CLICKHOUSE_DATASOURCE_USER=<readonly-username>
-e PMM_CLICKHOUSE_DATASOURCE_PASSWORD=<readonly-password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

Alternatively, you can use the `PMM_CLICKHOUSE_HOST` and `PMM_CLICKHOUSE_PORT` variables instead of `PMM_CLICKHOUSE_ADDR`.

```sh
-e PMM_CLICKHOUSE_HOST=<hostname>
-e PMM_CLICKHOUSE_PORT=<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_CLICKHOUSE_DATASOURCE_USER=<readonly-username>
-e PMM_CLICKHOUSE_DATASOURCE_PASSWORD=<readonly-password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

## Create the read-only data source user

Grafana runs data source queries on behalf of any signed-in user. This is the reason the ClickHouse data source must be accessed by a user with minimal privileges.

The built-in ClickHouse ships with a suitable read-only user. On an external ClickHouse, create an equivalent one and point `PMM_CLICKHOUSE_DATASOURCE_USER` and `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` at it:

```sql
CREATE USER <readonly-username> IDENTIFIED WITH sha256_password BY '<readonly-password>' SETTINGS readonly = 1, max_execution_time CHANGEABLE_IN_READONLY;
GRANT SELECT ON <database-name>.* TO <readonly-username>;
```

Use the same database name you pass in `PMM_CLICKHOUSE_DATABASE`, and grant nothing beyond `SELECT` on it.

`readonly = 1` blocks writes, DDL and all setting changes. The Grafana ClickHouse plugin sets `max_execution_time` on every query to enforce its query timeout, so that one setting is marked `CHANGEABLE_IN_READONLY`; without it, every query fails with `Cannot modify 'max_execution_time' setting in readonly mode`. Do not use `readonly = 2` instead — the [plugin documentation](https://grafana.com/docs/plugins/grafana-clickhouse-datasource/latest/configure/#clickhouse-user-and-permissions) advises against it.

`CHANGEABLE_IN_READONLY` requires the server to enable the following, otherwise ClickHouse rejects the constraint with `NOT_IMPLEMENTED`:

```xml
<access_control_improvements>
    <settings_constraints_replace_previous>true</settings_constraints_replace_previous>
</access_control_improvements>
```

If you restrict the user by network, allow the PMM Server host, since Grafana connects from there.

## Enhance ClickHouse security for PMM

When configuring PMM to use an external ClickHouse instance, make sure to enforce robust security practices to protect sensitive data and prevent unauthorized access:

- Enable SSL/TLS encryption for all connections
- Ensure that your ClickHouse instance is properly secured and monitored
- Disable empty passwords and plain text passwords
- Define all ClickHouse users explicitly, including permissions, to prevent automatic creation of unsecured users without passwords.
- Generate strong, random passwords for the dedicated PMM ClickHouse user. Use the following commands to generate a password and its SHA256 hash (useful for advanced ClickHouse configurations):

    ```sh
    PASSWORD=$(base64 < /dev/urandom | head -c12)
    echo "$PASSWORD" # note it down
    echo -n "$PASSWORD" | sha256sum | tr -d '-'
    ```

For more details, see the [ClickHouse user and roles settings](https://clickhouse.com/docs/operations/settings/settings-users).

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/guides/troubleshooting).

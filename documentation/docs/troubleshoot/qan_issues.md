# QAN issues

This section focuses on problems with QAN, such as queries not being retrieved.

## Missing data

### Why don't I see any query-related information?

There might be multiple places where the problem might come from:

- connection problem between pmm-agent and pmm-managed
- PMM-agent cannot connect to the database.
- data source is not properly configured.

### Why don't I see the whole query?

Long query examples and fingerprints are truncated to 2048 symbols by default to reduce space usage. In this case, the Explain section will not work. Max query size can be configured using flag `--max-query-length` while adding a service.

## Incorrect metrics: unrealistic query execution times 

If you're seeing query execution times that seem impossible (like 50,000+ seconds for simple SELECT statements), this is typically caused by metric calculation errors rather than actual performance issues. 

This is because enabling query plans causes `pg_stat_monitor` to create multiple records for each query, leading to incorrect timing calculations.

To fix the issue, disable query plan collection:

```sql
-- Check if query plan collection is enabled 
SHOW pg_stat_monitor.pgsm_enable_query_plan;

-- If it shows 'on', disable it 
ALTER SYSTEM SET pg_stat_monitor.pgsm_enable_query_plan = off;
SELECT pg_reload_conf();

-- Verify the change took effect
SHOW pg_stat_monitor.pgsm_enable_query_plan;
```

After disabling query plan collection, new metrics should show realistic execution times within minutes. 

## QAN service fails after upgrade

After upgrading PMM Server, the QAN service may fail to start with `BACKOFF`, `FATAL`, or `EXITED` status, preventing the QAN dashboard from loading. You'll see the following error in `/srv/logs/qan-api2.log`, where `x` is the migration version number:
```
stdlog: Migrations: Dirty database version x. Fix and force version.
```

This happens when the ClickHouse schema migration is interrupted during the upgrade.

### Resolution

- **PMM 3.5.0 and later:** The issue is **fixed automatically**. PMM detects and completes the interrupted schema migration upon restart.
- **Earlier versions:** Use the following manual workaround:
    {.power-number}

    1. Access the PMM container:
    ```bash
    podman exec -it pmm-server /bin/bash
    ```

    2. Connect to ClickHouse:
    ```bash
    clickhouse client --username=<clickhouse_user> --password=<clickhouse_password> -d pmm
    ```

    3. Fix the migration state. **Replace `x` with the version number from your error logs.**
    ```sql
    USE pmm;
    INSERT INTO schema_migrations (version, dirty, sequence) 
    VALUES (x, 0, toUnixTimestamp(NOW())*1000000000);
    EXIT;
    ```

    4. Restart QAN:
    ```bash
    supervisorctl restart qan-api2
    ```

    5. Verify QAN is running:
    ```bash
    supervisorctl status qan-api2
    ```

## ClickHouse memory issues in low-memory environments

If you're running PMM Server with less than 16 GB RAM and seeing "memory limit exceeded" errors in ClickHouse logs, switch to the low-memory configuration.

PMM includes two ClickHouse profiles:

- **default**: optimized for performance (16 GB+ RAM)
- **low-memory**: optimized for constrained environments, based on [ClickHouse recommendations](https://clickhouse.com/docs/operations/tips#using-less-than-16gb-of-ram)

### Switch to low-memory configuration

```bash
docker exec -it -u pmm pmm-server ./switch-config.sh low
```

To switch back:
```bash
docker exec -it -u pmm pmm-server ./switch-config.sh default
```

The script stops ClickHouse, updates the configuration, and restarts the service.

### Persistent configuration

If you run PMM Server with the `--rm` flag, run the switch script each time the container starts. For systemd, add to your unit file:
```ini
ExecStartPost=/usr/bin/docker exec -u pmm pmm-server ./switch-config.sh low
```

See [Install PMM Server with Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md) for systemd configuration examples.

!!! note "Configuration details"
    Both configuration files are located in `/etc/clickhouse-server/` inside the PMM Server container:
    
    - `default-config.xml`: default profile
    - `low-memory-config.xml`: low-memory profile
    
    The script is available at `/etc/clickhouse-server/switch-config.sh` or `/opt/switch-config.sh`.
    
    When switching profiles, both `config.xml` and `users.xml` are updated to point to the selected profile.
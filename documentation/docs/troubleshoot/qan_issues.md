
# QAN issues

This section focuses on problems with QAN, such as queries not being retrieved so on.

## Missing data

### Why don't I see any query-related information?

There might be multiple places where the problem might come from:

- Connection problem between pmm-agent and pmm-managed
- PMM-agent cannot connect to the database.
- Data source is not properly configured.

### Why don't I see the whole query?

Long query examples and fingerprints is truncated to 2048 symbols by default to reduce space usage. In this case, the query explains section will not work. Max query size can be configured using flag `--max-query-length` while adding a service.

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

After upgrading PMM Server, the QAN service may fail to start with `BACKOFF`, `FATAL`, or `EXITED` status, preventing the QAN dashboard from loading. You'll see the following error in `/srv/logs/qan-api2.log`:
```
stdlog: Migrations: Dirty database version x. Fix and force version.
```
This happens when the ClickHouse schema migration is interrupted during the upgrade.

**Resolution:**

- **PMM 3.5.0 and later:** Fixed automatically. PMM detects and completes the interrupted schema migration.
- **Earlier versions:** Use the manual workaround below:
{.power-number}

    1. Access PMM container:
    ```bash
    podman exec -it pmm-server /bin/bash
    ```

    2. Connect to ClickHouse:
    ```bash
    clickhouse-client
    ```

    3. Fix the migration state:
    ```sql
    USE pmm;
    INSERT INTO schema_migrations (version, dirty, sequence) 
    VALUES (18, 0, toUnixTimestamp(NOW())*1000000000);
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
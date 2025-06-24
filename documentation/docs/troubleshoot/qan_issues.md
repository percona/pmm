
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

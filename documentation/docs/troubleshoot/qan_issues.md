
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

The most common cause is the `pg_stat_monitor.pgsm_enable_query_plan setting` in PostgreSQL with `pg_stat_monitor extension`, which has a known issue where it interferes with time metric calculations. When enabled, the extension reports timing data in microseconds, but PMM interprets it as milliseconds, resulting in times that are off by a factor of 1000 or more. 

This causes:

- Query execution times showing 10,000+ seconds for queries that should take milliseconds
- Times that are exactly 1000x higher than expected
- All queries showing unreasonably high execution times consistently

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

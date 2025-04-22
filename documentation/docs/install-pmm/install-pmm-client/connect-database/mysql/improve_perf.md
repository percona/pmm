# Optimize MySQL monitoring performance in PMM

When monitoring MySQL instances with a large number of tables, PMM's data collection can impact both client and database performance. Here are a few optimization options to ensure efficient monitoring without overloading your systems.

## Options for table statistics optimization

PMM provides two command-line options when adding MySQL instances to control table statistics collection:

- `--disable-tablestats`- Completely disables table statistics collection when there are more than 1000 tables (the default limit).
- `--disable-tablestats-limit`- Customizes the threshold (number of tables) at which table statistics collection is disabled

### When to use these options

Consider using these options in the following when:

-  monitoring MySQL instances with thousands of tables
-  you notice high resource usage on either PMM Client or your MySQL server
-  you observe monitoring delays or timeouts during data collection

### Disable table statistics

For MySQL instances with many tables, you can completely disable per-table statistics collection:
```sh
pmm-admin add mysql --disable-tablestats
```

This command configures PMM to: 

- add your MySQL instance to PMM without collecting table-level statistics
- still collect all instance-level and database-level metrics
- significantly reduce the monitoring load when you have more than 1000 tables

##  Set a custom table limit
For more precise control, you can specify a custom limit for when table statistics should be disabled:


# Change the number of tables

When adding an instance with `pmm-admin add`, the `--disable-tablestats-limit` option changes the number of tables (from the default of 1000) beyond which per-table statistics collection is disabled:

```sh
pmm-admin add mysql --disable-tablestats-limit=<LIMIT>
```

This command configures PMM to: 

- collect table statistics normally until the instance reaches 2000 tables
- automatically disable table statistics when the number of tables exceeds 2000
- continue collecting all other MySQL metrics normally

## Best practices for performance optimization

To finding the right balance:

- If you have more than 1000 tables, begin with `--disable-tablestats` to start conservative
- Check CPU, memory, and network usage on the client to monitor PMM Client resource usage.
- Watch for increased load during monitoring intervals to monitor MySQL load. 
- If resources permit, you can try enabling table statistics with a higher limit and adjust incrementally.

Additional performance considerations: 

- For high-traffic MySQL servers, consider using query sampling with the slow log. For details, see  [MySQL data source configuration](../mysql/mysql.md#slow-query-log-configuration).
- Adjust metrics collection frequency for remote instances.  For details, see [Remote instances monitoring](../remote.md#recommended-settings)
- Ensure PMM Client has adequate CPU and memory resources on busy database servers

## Change settings after initial setup

These settings only apply when adding an instance with `pmm-admin add`. Only one of the table statistics options can be used when adding an instance.

To change them after initial setup:
{.power-number}

1. Remove the existing MySQL service:
```sh
pmm-admin remove mysql SERVICE_NAME
```
2. Add the service again with the desired table statistics settings:
```sh
pmm-admin add mysql --disable-tablestats-limit=3000 [OTHER_OPTIONS] SERVICE_NAME
```

## Performance impact comparison

| Scenario                       | Table stats enabled        | Table stats disabled                    |
|-------------------------------|----------------------------|------------------------------------------|
| Small MySQL (< 100 tables)    | Minimal impact             | Not necessary                            |
| Medium MySQL (100–1000 tables)| Moderate impact            | Minimal performance gain                 |
| Large MySQL (1000–5000 tables)| High impact                | Significant performance improvement      |
| Very large MySQL (> 5000 tables)| Severe impact            | **Strongly recommended**                 |

## Related Topics

- [MySQL connection options](../mysql/mysql.md)
- [Performance Schema vs. Slow Query Log](../mysql/mysql.md#performance-schema-configuration)
- [Configure metrics resolution](../../../../configure-pmm/metrics_res.md)

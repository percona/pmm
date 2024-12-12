
# QAN issues

This section focuses on problems with QAN, such as queries not being retrieved so on.

## Missing data

**Why don't I see any query-related information?**

There might be multiple places where the problem might come from:

- Connection problem between pmm-agent and pmm-managed
- PMM-agent cannot connect to the database.
- Data source is not properly configured.


**Why don't I see the whole query?**

Long query examples and fingerprints can be truncated to 1024 symbols to reduce space usage. In this case, the query explains section will not work.
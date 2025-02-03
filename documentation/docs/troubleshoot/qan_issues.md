
# QAN issues

This section focuses on problems with QAN, such as queries not being retrieved so on.

## Missing data

**Why don't I see any query-related information?**

There might be multiple places where the problem might come from:

- Connection problem between pmm-agent and pmm-managed
- PMM-agent cannot connect to the database.
- Data source is not properly configured.


**Why don't I see the whole query?**

Long query examples and fingerprints is truncated to 2048 symbols by default to reduce space usage. In this case, the query explains section will not work. Max query size can be configured using flag `--max-query-length` while adding a service.

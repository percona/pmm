# MySQL Query Response Time Details

![!image](../../images/PMM_MySQL_Query_Response_Time_Details.jpg)

## Average Query Response Time

The Average Query Response Time graph shows information collected using the Response Time Distribution plugin sourced from [table `INFORMATION_SCHEMA.QUERY_RESPONSE_TIME`](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME). It computes this value across all queries by taking the sum of seconds divided by the count of queries.

## Query Response Time Distribution

Query response time counts (operations) are grouped into three buckets:

- 100 ms - 1 s

- 1 s - 10 s

- &gt; 10 s

## Average Query Response Time

Available only in [Percona Server for MySQL](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#logging-the-queries-in-separate-read-and-write-tables), provides  visibility of the split of [READ](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_READ) vs [WRITE](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#QUERY_RESPONSE_TIME_WRITE) query response time.

## Read Query Response Time Distribution

Available only in Percona Server for MySQL, illustrates READ query response time counts (operations) grouped into three buckets:

- 100 ms - 1 s

- 1 s - 10 s

- &gt; 10 s

## Write Query Response Time Distribution

Available only in Percona Server for MySQL, illustrates WRITE query response time counts (operations) grouped into three buckets:

- 100 ms - 1 s

- 1 s - 10 s

- &gt; 10 s

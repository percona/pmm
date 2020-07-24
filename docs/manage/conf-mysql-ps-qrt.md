# Query Response Time Plugin

Query response time distribution is a feature available in Percona Server.  It
provides information about changes in query response time for different groups
of queries, often allowing to spot performance problems before they lead to
serious issues.

To enable collection of query response time:


1. Install the `QUERY_RESPONSE_TIME` plugins:

```
INSTALL PLUGIN QUERY_RESPONSE_TIME_AUDIT SONAME 'query_response_time.so';
INSTALL PLUGIN QUERY_RESPONSE_TIME SONAME 'query_response_time.so';
INSTALL PLUGIN QUERY_RESPONSE_TIME_READ SONAME 'query_response_time.so';
INSTALL PLUGIN QUERY_RESPONSE_TIME_WRITE SONAME 'query_response_time.so';
```


2. Set the global varible `query_response_time_stats` to `ON`:

```
SET GLOBAL query_response_time_stats=ON;
```

**See also**


* [Percona Server 5.7: query_response_time_stats](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#query_response_time_stats)


* [Percona Server 5.7: Response time distribution](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#installing-the-plugins)

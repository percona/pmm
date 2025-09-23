# PostgreSQL log_autovacuum_min_duration is disabled (Set to -1)

## Description

Enable logging of autovacuum run information for useful information about autovacuum with almost no drawbacks.

Information about autovacuum (and autoanalyze) executions in the logs is extremely useful and provides a detailed history of dead row garbage collection on the PostgreSQL instance, and may show signs of misconfiguration. 

It is a cheap way of augmenting the regular monitoring based on views, and the only downside is that the volume of logs will be increased slightly.

## Resolution

Set **log_autovacuum_min_duration** server configuration option to the value of **0** or more. 

Value >0 will mean that only runs taking longer than so many milliseconds are logged. That can be done online, and the change will reflect immediately. 

Next autovacuum or autoanalyze run information will be present in the PostgreSQL logs.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

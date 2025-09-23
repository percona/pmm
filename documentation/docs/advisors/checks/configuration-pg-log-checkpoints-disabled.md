# PostgreSQL log_checkpoints is not enabled

## Description

It is recommended to enable the logging of checkpoint information, as that provides a lot of useful information with almost no drawbacks.

Information about checkpoints in the logs is extremely useful and provides a detailed history of changes in write load on the PostgreSQL instance. 

In addition, it gives insight into the IO performance. It is a cheap way of augmenting the regular monitoring based on views, and the only downside is that the volume of logs will be increased slightly.

## Resolution

Set **log_checkpoints** server configuration option to **ON**. You can do this online, and the change will reflect immediately. 

Next checkpoint information will be present in the PostgreSQL logs.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

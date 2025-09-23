# MongoDB active vs available connections

## Description
This check returns a warning if the ratio between active and available connections is higher than 75%. 

This check is relevant because of the risk of running out of connections. If this happens, application won't be able to connect to MongoDB.

## Resolution
We recommend increasing the ULIMIT to accept more connections or to evaluate the current workload from applications.

An unexpected spike on the workload or not optimized queries could be the root cause of more active connections.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

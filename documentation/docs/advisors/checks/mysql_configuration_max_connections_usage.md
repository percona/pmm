# MySQL max connection usage check

## Description

The status variable **max_used_connections** indicates the highest number of connections used since the previous system restart. This warrants attention when the value approaches the predetermined **max_connections** limit.

## Resolution

Revisit the **max_connections** configuration option and revise it inside the limit of your platform resources, if necessary. Alternatively, manage your max connection usage.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

# PostgreSQL max_connections set too high

## Description

PostgreSQL doesn't cope well with having many connections, even if they are idle. 

The recommended value is below 300.

Even if there are currently fewer connections than the **max_connections** value configured, the recommendation is to put a hard limit. 

Connection spikes and new applications will eventually move the number of connections higher than an acceptable threshold. 



 a significant number of connections is required, a pooling solution should be used.

This limitation comes from the fact that PostgreSQL maintains snapshots for each connection. Each new transaction will have to perform operations on the snapshots, and the more connections (and thus snapshots) there are, the higher the impact on TPS. 
 
## Resolution

Review the number of connections that applications require at peak. 

 If this number is below the recommended value of 300, adjust **max_connections** to 300. 



 If the peaks are higher than the recommended value, consider adjusting the way applications are using the database. 



 If application-side pooling is used, allocate fewer connections per application. 

If no pooling is available for the application side, a middleware pooler like PgBouncer should be considered.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

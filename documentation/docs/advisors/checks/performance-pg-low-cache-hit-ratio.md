# PostgreSQL cache hit ratio

## Description

In an ideal, healthy, and well-configured PostgreSQL environment, the hit ratio metric should be above 90%. This means that every 9 out of 10 reads are successfully resolved from in-memory reads rather than performing IO access to the disk. This produces a higly performant environment. 

When the hit ratio metric value falls below 80% consistently, this can be due to a misconfigured instance in relation to the workload. Typically, the cache is not big enough to keep the most accessed data in memory.

## Resolution

For PostgreSQL databases, the cache buffer size is configured with the shared_buffer configuration.

It tells the database how much of the machine’s memory can allocate for storing data in memory. 

The default is very low since it is set to only 128 MB. The optimal value ultimately depends on the data access patterns for a given database, but PostgreSQL recommends that you should initially configure it to 25% of the available total memory if you are using a dedicated database server. 

As a first remediation action, you can check that the shared_buffer cache is well-configured in relation to the machine's memory.

In some other scenarios, the application can be accessing historical data, which is rarely visited. Therefore, it is not present in memory so the disk reads are required. 

Here the recommendation can involve setting up a separate system for the historical data to avoid impacting the OLTP workload.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
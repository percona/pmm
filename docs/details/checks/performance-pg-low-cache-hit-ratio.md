# PostgreSQL Cache Hit Ratio

## Description

In an ideal, healthy and well configured PostgreSQL environment the hit ratio metric should be above 90%, which means each 9 from 10 reads are successfully resolved from in-memory reads rather than performing IO access to the disk, this produces a very performant environment. 

When the hit ratio metric value gets below the 80% in a consistent way it can be related to a misconfigured instance in relation to the workload, typically the cache is not big enough to keep the most accessed data in memory.

## Resolution

For PostgreSQL databases, the cache buffer size is configured with the shared_buffer configuration. It tells the database how much of the machine’s memory it can allocate for storing data in memory. The default is very low since it is set to only128 MB, the optimal value ultimately depends on the data access patterns for a given database, but PostgreSQL recommends that you should initially configure it to 25% of the available total memory if you are using a dedicated database server. 

As the first remediation action you can verify if the shared_buffer cache is well configured in relation to the machine's memory.

In some other scenarios the application can be accessing historical data, which is rarely visited, therefore is not present in memory so the disk reads are required, here the recommendation can involve the setup of a separate system for the historical data to avoid impacting the OLTP workload.
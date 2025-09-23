# Check WiredTiger cache size

## Description
This advisor warns if the configured WiredTiger cache size is greater than 50% of server memory. 

This is important because MongoDB uses the remaining available memory (filesystem cache) beyond its own storage engine cache to maintain connections, aggregation/sort operations, cursors, etc. 

Keeping a non-default high cache size for WiredTiger can cause OOM (Out of memory) issues.

To avoid performance issues, see the [WiredTiger Storage Engine blog](https://www.mongodb.com/docs/manual/core/wiredtiger/#memory-use) to check and set the right value for your environment.


## Rule

{% raw %}
```
METRICS_INSTANT
# to fetch the value
node_memory_numa_MemTotal{node_name="{{.NodeName}}"}

METRICS_INSTANT
# to fetch the value
mongodb_ss_wt_cache_maximum_bytes_configured{service_name="{{.ServiceName}}"}            
```
{% endraw %}

## Resolution
- Let WiredTiger use the default cache size of 50% of (RAM - 1 GB), OR 
- Make sure that the **storage.wiredTiger.engineConfig.cacheSizeGB** value in the configuration file doesn’t exceed 50% of the server memory.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
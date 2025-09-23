# MongoDB profiling level is too high 

## Description
This advisor warns if the global profiling level is too high - it is set to anything other than 0.

Setting the profiler to gather information at all times is not recommended because profiling has an effect on the performance and disk use. 

using profiling in peak periods with high traffic workloads where the database has to collect data for all the operations is dangerous. This can add overhead that results in performance degradation - especially in very active environments.

When necessary, use the profiler in off-peak periods with less workload. It is always recommended to only enable profiling (Level 1) for short periods of time in order to perform any needed query analysis or troubleshooting. 

To avoid performance degradation, avoid setting and using the profiler for long periods of time in production environments.

For more information, see [db.setProfilingLevel in the MongoDB documentation](https://docs.mongodb.com/manual/reference/method/db.setProfilingLevel/).


## Rule
```
operationProfiling.mode
profiling = parsed.get("operationProfiling", {})
            profiler = (profiling.get("mode")
```
Can also be queried from within the database via the following commands:

- __db.getProfilingLevel();__

- __db.getProfilingStatus();__


## Resolution
Turn off profiler or reduce the level. You can do this either from the command line startup or via the config file.

To turn off profiler level globally:
1. Edit the **mongod.conf** file and disable/comment below parameter: 
   **operationProfiling**
   OR, adjust the **mode**: 

``` 
operationProfiling:
     mode: off
```


2. Roll-restart your mongod nodes.


To reduce the profiler level, edit and adjust the following parameter in **mongod.conf**:

``` 
	operationProfiling:
   		mode: slowOp
```

To check the profiler status and level on a running system, run the following commands from the mongo shell:

```
		db.getProfilingStatus()
		db.getProfilingLevel()

Note: We can include different options here to manipulate collection of operations.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

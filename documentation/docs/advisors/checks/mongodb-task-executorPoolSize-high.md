# Check taskExecutorPoolsize value

## Description
This advisor warns if the number of Task Executor connection pools (the **taskExecutorPool** value) is higher than the number of available CPU cores of a server. 

This is relevant because the performance can drop if the number of task executor pools set is too high.

The parameter can be set in the configuration file, OR using the **setParameter** command during runtime. 

Keep in mind that if set at runtime, the parameter setting will not persist after the server restart, unless it is also specified in the `mongodb.conf`  configuration file. 

To avoid performance issues, check and set the right value for your environment based on [MongoDB documentation](https://www.mongodb.com/docs/manual/reference/parameters/#mongodb-parameter-param.taskExecutorPoolSize).   

## Rule

{% raw %}
```MONGODB_GETPARAMETER
# to fetch the value
db.adminCommand( { ‘getParameter’: ‘*’  } ).taskExecutorPoolSize

METRICS_INSTANT
# to fetch the value
mongodb_sys_cpu_num_cpus{service_name="{{.ServiceName}}"}

``` 
{% endraw %}

## Resolution
Adjust the value of this metric to match what is available on the host. Alternatively, consider an upgrade to add more CPU cores if your application needs exceed the currently allocated resources.

These settings apply only to your `mongos` nodes.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
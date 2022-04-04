# MongoDB Profiling level set to high 

## Description
This check returns a notice/warning if the global profiling level is set too high - anything other than 0. Having the profiler set to gather information at all times is not recommended. 
This is dangerous in busy and high traffic workloads where the database has to collect data for all the operations. This can add overhead that results in performance degradation - especially in very active environments.

It is always recommended to only enable profiling (Level 1) for short periods of time in order to perform any needed query analysis or troubleshooting.  When necessary,  the profiler should be used during less busy windows and then turned off.  It is not recommended to set and use the profiler for long periods of time in production environments due to performance degradation
[https://docs.mongodb.com/manual/reference/method/db.setProfilingLevel/](https://docs.mongodb.com/manual/reference/method/db.setProfilingLevel/)


## Rule
```
operationProfiling.mode
profiling = parsed.get("operationProfiling", {})
            profiler = (profiling.get("mode")
```
Can also be queried from within the database via the following commands:

__db.getProfilingLevel();__

__db.getProfilingStatus();__


## Resolution
Please Perform the steps mentioned below to turn off profiler completely or reduce the level

The profiler can be enabled or have the level changed at either the command line startup or via the config file.


1. To turn off profiler level globally, edit the mongod.conf file and disable/comment below parameter.\
   “operationProfiling”\
   OR, adjust the “mode”\
   operationProfiling:\
     mode: off
2. Then Perform a rolling restart of your mongod nodes
3. To reduce the profiler level, edit and adjust below parameter in mongod.conf
	operationProfiling:
   		mode: slowOp
4. To check the profiler status and level on a running system, performa the following commands from the mongo shell:
		db.getProfilingStatus()
		db.getProfilingLevel()


Note: We can include different options here to manipulate collection of operations.

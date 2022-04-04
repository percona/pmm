# MongoDB logLevel is not default

## Description

This check returns a warning if the logLevel is not default. This can cause the disks to fill up.
The logLevel can be set as an integer between 0 and 5, where 5 is the most verbose.
The default logLevel is 0 (Informational).
The logLevels can be adjusted for your **mongod** and your **mongos** nodes.

It is recommended to use the default logLevel to avoid excessive disk usage.
Increasing the verbosity of log levels is useful for debugging purposes for a short period of time.
[https://docs.mongodb.com/manual/reference/method/db.setLogLevel/](https://docs.mongodb.com/manual/reference/method/db.setLogLevel/)

## Rule

MONGODB_GETPARAMETER

`db.adminCommand( { getParameter: 1, "logLevel": 1 } )`

## Resolution

Please Perform the steps mentioned below to turn on or adjust the verbosity of your logs.

It is possible to do it online:

`mongo> db.setLogLevel(1);`

Or using the adminCommand syntax
`db.adminCommand( { setParameter: 1, logLevel: 2 } )`

1. Set to default. \
   Edit mongod.conf and set the below parameter.

```
      setParameter:
       logLevel: 0
```

2. If resetting the log level in your mongod config file, be aware that this will not take effect until the next restart.

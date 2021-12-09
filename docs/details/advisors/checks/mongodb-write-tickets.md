# MongoDB write ticket is more than 128.

## Description
This check returns a warning if the write ticket is more than 128. This can cause performance issues.
Ideally the number of tickets should be based on the number of CPU available.
The default write ticket is 128.
It can be adjusted for your mongod and your mongos nodes.

[https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentWriteTransactions](https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentWriteTransactions)


## Rule
MONGODB_GETPARAMETER

`db.adminCommand( { setParameter: 1, "wiredTigerConcurrentWriteTransactions": "128"  } )`

## Resolution
Please Perform the steps mentioned below to turn adjust the verbosity of your logs.

It is possible to do it online:

`mongo> db.adminCommand( { setParameter: 1, "wiredTigerConcurrentWriteTransactions": "128"  } )`

1. Set to default. \
Edit mongod.conf and set the below parameter.
```
       setParameter:
         wiredTigerConcurrentWriteTransactions: 128
```
2. If resetting the write ticket in your mongod config file, be aware that this will not take effect until the next restart.

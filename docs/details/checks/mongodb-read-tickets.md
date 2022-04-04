# MongoDB read ticket is more than 128

## Description
This check returns a warning if the read ticket is more than 128. This can cause performance issues.
Ideally the number of tickets should be based on the number of CPU available.
The default read ticket is 128.
It can be adjusted for your mongod and your mongos nodes.

[https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentReadTransactions](https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentReadTransactions)



## Rule
MONGODB_GETPARAMETER

`db.adminCommand( { setParameter: 1, "wiredTigerConcurrentReadTransactions": "128"  } )`

## Resolution
Please Perform the steps mentioned below to turn adjust the verbosity of your logs.

It is possible to do it online:

`mongo> db.adminCommand( { setParameter: 1, "wiredTigerConcurrentReadTransactions": "128"  } );`

1. Set to default. 
Edit mongod.conf and set the below parameter.
```
          setParameter:
            wiredTigerConcurrentReadTransactions: 256
```
2. If resetting the read ticket in your mongod config file, be aware that this will not take effect until the next restart.

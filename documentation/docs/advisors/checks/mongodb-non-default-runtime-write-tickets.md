# Write tickets during runtime are > 128

## Description
This advisor warns if the number of write transactions (tickets) into the WiredTiger storage during runtime is more than 128. 

This is relevant because the performance can drop if the number of write tickets is too high.

Ideally, the number of tickets should be based on the number of CPU available. 
The default number of write tickets is 128. You can adjust it for both your `mongod` and `mongos` nodes. 

See [wiredTigerConcurrentWriteTransactions](https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentWriteTransactions) in the MongoDB documentation.

This parameter also needs to be set at the configuration file.


## Rule 
``` MONGODB_GETPARAMETER
db.adminCommand( { setParameter: 1, "wiredTigerConcurrentWriteTransactions": "128"  } ) 
```
 
## Resolution
Adjust the number of write tickets allowed into the WiredTiger storage engine. 

* Using the `setParameter` shell helper:

   ```
   mongo> db.adminCommand( { setParameter: 1, "wiredTigerConcurrentWriteTransactions": "128"  } )
   ```

* Editing the configuration file 

   ``` yaml
   setParameter:     
      wiredTigerConcurrentWriteTransactions: 128
   ``` 

   Note that the changes in the configuration file will take effect only after the server restart.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

# MongoDB read ticket is more than 128

## Description
This advisor warns if the number of concurrent read transactions (tickets) into the WiredTiger storage is more than 128. If the number is too high, this  can cause performance issues.

Ideally, the number of read tickets should be based on the number of CPU available. 

The default number of read tickets is 128. You can adjust it for both your `mongod` and `mongos` nodes.

For more information, see [wiredTigerConcurrentReadTransactions in the MongoDB documentation](https://docs.mongodb.com/manual/reference/parameters/#mongodb-parameter-param.wiredTigerConcurrentReadTransactions).



## Rule
MONGODB_GETPARAMETER

`db.adminCommand( { setParameter: 1, "wiredTigerConcurrentReadTransactions": "128"  } )`

## Resolution
TAdjust the number of write tickets allowed into the WiredTiger storage engine. 

* Using the `setParameter` shell helper:
   
   ```
   mongo> db.adminCommand( { setParameter: 1, "wiredTigerConcurrentReadTransactions": "128"  } );
   ```

* Editing the configuration file: 

   ``` yaml
   setParameter:
      wiredTigerConcurrentReadTransactions: 256
   ```

   Note that the changes in the configuration file will take effect only after the server restart.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
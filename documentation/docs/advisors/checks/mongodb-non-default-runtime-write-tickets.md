# MongoDB write tickets at runtime

The `mongodb_write_tickets_runtime` check warns if the number of write tickets that MongoDB uses at runtime isn't the recommended one.

## Description

Write tickets limit the number of concurrent write transactions into the storage engine. The `storageEngineConcurrentWriteTransactions` parameter sets the number of write tickets. Before MongoDB 6.0, the parameter was named `wiredTigerConcurrentWriteTransactions`, and later versions keep the old name as an alias.

The check reads the parameter with the `getParameter` command and the server version with the `buildInfo` command. The rule depends on the MongoDB version:

- **MongoDB 7.0 and later**: MongoDB adjusts the number of write tickets dynamically, up to 128. Setting the parameter to any value other than `0` replaces the dynamic adjustment with a fixed number of tickets, so the check warns about any such value.
- **Earlier versions**: The default number of write tickets is 128. The check warns if the number is more than 128, because performance can drop if the number of write tickets is too high. Ideally, the number of tickets is based on the number of available CPUs.

For details, see [storageEngineConcurrentWriteTransactions](https://www.mongodb.com/docs/manual/reference/parameters/#mongodb-parameter-param.storageEngineConcurrentWriteTransactions) in the MongoDB documentation. For the value in the startup options, see [MongoDB write tickets in the startup options](mongodb-write-tickets.md).

## Rule
``` MONGODB_GETPARAMETER
db.adminCommand( { getParameter: "*" } ).storageEngineConcurrentWriteTransactions
```

## Resolution

For MongoDB 7.0 and later, remove the parameter from the `setParameter` section of the configuration file and from the command line, then restart `mongod`. Keep the parameter only if MongoDB Support recommended it.

For earlier versions, set the number of write tickets to 128 or lower:

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

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

# MongoDB logLevel is not default

## Description

This advisor warns if the verbosity level of MongoDB logs is higher than the default value. 

The verbosity level is controlled by the `logLevel` parameter. Its value is an integer between 0 and 5, where `0` is the default log level (Informational) and `5` is the most verbose level. 

Increasing the verbosity of log messages is useful for debugging purposes for a short period of time.

For more information, see [db.setLogLevel() in the MongoDB documentation](https://docs.mongodb.com/manual/reference/method/db.setLogLevel/).


## Rule
MONGODB_GETPARAMETER

`db.adminCommand( { getParameter: 1, "logLevel": 1 } )`

## Resolution

Turn on or adjust the verbosity of your logs: 

=== "From the `mongo`/`mongosh` shell"

     - Using the `db.setLogLevel()` method: 

         ```
         db.setLogLevel(1);  
         ```

     - using the `adminCommand` syntax: 
       
       ```
       db.adminCommand( { setParameter: 1, logLevel: 2 } )
       ```

=== "In the configuration file" 

     Set the following parameter to default:
     ```yaml
           setParameter:
            logLevel: 0
     ``` 

     Restart the `mongod` nodes for the changes to take effect. Use the rolling restart method for your cluster or replica set. 

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
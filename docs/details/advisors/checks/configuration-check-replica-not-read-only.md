# MySQL Replica node is not configured as READ-ONLY

## Description
A node that works as a replica must also have the READ-ONLY flag active. This is to prevent accidental writes on the node which may lead to data inconsistency.
The current node has a READ-ONLY value of 0, as such the node is at high risk. 



## Rule
`SELECT @@global.read_only, performance_schema.replication_connection_configuration;`


## Resolution
Set the value of READ-ONLY to 1, to prevent writes on this node.
`SET GLOBAL READ-ONLY=1;`
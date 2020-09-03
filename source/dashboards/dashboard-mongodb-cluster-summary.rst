#######################
MongoDB Cluster Summary
#######################

.. image:: /_images/PMM_MongoDB_Cluster_Summary.jpg

*****************************
Current Connections Per Shard
*****************************

TCP connections (Incoming) in mongod processes.

*****************
Total Connections
*****************

Incoming connections to mongos nodes.

*****************
Cursors Per Shard
*****************

The Cursor is a MongoDB Collection of the document which is returned upon the find method execution.

**************
Mongos Cursors
**************

The Cursor is a MongoDB Collection of the document which is returned upon the find method execution.

********************
Operations Per Shard
********************

Ops/sec, classified by legacy wire protocol type (query, insert, update, delete, getmore).

***********************
Total Mongos Operations
***********************

Ops/sec, classified by legacy wire protocol type (query, insert, update, delete, getmore).

********************
Collection Lock Time
********************

MongoDB uses multi-granularity locking that allow shared access to a resource, such as a database or collection.

*****************
Change Log Events
*****************

Count, over last 10 minutes, of all types of config db changelog events.

******************
Oplog Range by Set
******************

Timespan 'window' between oldest and newest ops in the Oplog collection.

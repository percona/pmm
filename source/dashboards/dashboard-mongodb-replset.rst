.. _dashboard-mongodb-replset:

###############
MongoDB ReplSet
###############

This dashboard provides information about replica sets and their members.


.. _dashboard-mongodb-replset.replication-operations:
.. _replication-operations:

**********************
Replication Operations
**********************

This metric provides an overview of database replication operations by type and
makes it possible to analyze the load on the replica in more granular
manner. These values only appear when the current host has replication enabled.

.. _dashboard-mongodb-replset.replset-state:
.. _replset-state:

*************
ReplSet State
*************

This metric shows the role of the selected member instance (PRIMARY or SECONDARY).

.. _dashboard-mongodb-replset.replset-members:
.. _replset-members:

***************
ReplSet Members
***************

This metric the number of members in the replica set.

.. _dashboard-mongodb-replset.replset-last-election:
.. _replset-last-election:

*********************
ReplSet Last Election
*********************

This metric how long ago the last election occurred.

.. _dashboard-mongodb-replset.replset-lag:
.. _replset-lag:

***********
ReplSet Lag
***********

This metric shows the current replication lag for the selected member.

.. _dashboard-mongodb-replset.storage-engine:
.. _storage-engine:

**************
Storage Engine
**************

This metric shows the storage engine used on the instance

.. _dashboard-mongodb-replset.oplog-insert-time:
.. _oplog-insert-time:

*****************
Oplog Insert Time
*****************

This metric shows how long it takes to write to the oplog. Without it the write
will not be successful.

This is more useful in mixed replica sets (where instances run different storage
engines).

.. _dashboard-mongodb-replset.oplog-recovery-window:
.. _oplog-recovery-window:

*********************
Oplog Recovery Window
*********************

This metric shows the time range in the oplog and the oldest backed up
operation.

For example, if you take backups every 24 hours, each one should contain at
least 36 hours of backed up operations, giving you 12 hours of restore window.

.. _dashboard-mongodb-replset.replication-lag:
.. _replication-lag:

***************
Replication Lag
***************

This metric shows the delay between an operation occurring on the primary and
that same operation getting applied on the selected member

.. _dashboard-mongodb-replset.elections:
.. _elections:

*********
Elections
*********

Elections happen when a primary becomes unavailable. Look at this graph over
longer periods (weeks or months) to determine patterns and correlate elections
with other events.

.. _dashboard-mongodb-replset.member-state-uptime:
.. _member-state-uptime:

*******************
Member State Uptime
*******************

This metric shows how long various members were in PRIMARY and SECONDARY roles.

.. _dashboard-mongodb-replset.max-heartbeat-time:
.. _max-heartbeat-time:

******************
Max Heartbeat Time
******************

This metric shows the heartbeat return times sent by the current member to other
members in the replica set.

Long heartbeat times can indicate network issues or that the server is too busy.

.. _dashboard-mongodb-replset.max-member-ping-time:
.. _max-member-ping-time:

********************
Max Member Ping Time
********************

This metric can show a correlation with the replication lag value.

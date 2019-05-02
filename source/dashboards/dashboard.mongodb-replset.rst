.. _dashboard.mongodb-replset:

|mongodb| ReplSet
================================================================================

This dashboard provides information about replica sets and their members.

.. contents::
   :local:

.. _dashboard.mongodb-replset.replication-operations:

`Replication Operations <dashboard.mongodb-replset.html#replication-operations>`_
----------------------------------------------------------------------------------

This metric provides an overview of database replication operations by type and
makes it possible to analyze the load on the replica in more granular
manner. These values only appear when the current host has replication enabled.

.. _dashboard.mongodb-replset.replset-state:

`ReplSet State <dashboard.mongodb-replset.html#replset-state>`_
--------------------------------------------------------------------------------

This metric shows the role of the selected member instance (PRIMARY or SECONDARY).

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.replset-members:

`ReplSet Members <dashboard.mongodb-replset.html#replset-members>`_
--------------------------------------------------------------------------------

This metric the number of members in the replica set.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.replset-last-election:

:ref:`ReplSet Last Election <dashboard.mongodb-replset.html#replset-last-election>`
--------------------------------------------------------------------------------

This metric how long ago the last election occurred.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.replset-lag:

`ReplSet Lag <dashboard.mongodb-replset.html#replset-lag>`_
--------------------------------------------------------------------------------

This metric shows the current replication lag for the selected member.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.storage-engine:

`Storage Engine <dashboard.mongodb-replset.html#storage-engine>`_
--------------------------------------------------------------------------------

This metric shows the storage engine used on the instance

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.oplog-insert-time:

`Oplog Insert Time <dashboard.mongodb-replset.html#oplog-insert-time>`_
--------------------------------------------------------------------------------

This metric shows how long it takes to write to the oplog. Without it the write
will not be successful.

This is more useful in mixed replica sets (where instances run different storage
engines).

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.oplog-recovery-window:

`Oplog Recovery Window <dashboard.mongodb-replset.html#oplog-recovery-window>`_
--------------------------------------------------------------------------------

This metric shows the time range in the oplog and the oldest backed up
operation.

For example, if you take backups every 24 hours, each one should contain at
least 36 hours of backed up operations, giving you 12 hours of restore window.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.replication-lag:

`Replication Lag <dashboard.mongodb-replset.html#replication-lag>`_
--------------------------------------------------------------------------------

This metric shows the delay between an operation occurring on the primary and
that same operation getting applied on the selected member

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.elections:

`Elections <dashboard.mongodb-replset.html#elections>`_
--------------------------------------------------------------------------------

Elections happen when a primary becomes unavailable. Look at this graph over
longer periods (weeks or months) to determine patterns and correlate elections
with other events.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.member-state-uptime:

`Member State Uptime <dashboard.mongodb-replset.html#member-state-uptime>`_
--------------------------------------------------------------------------------

This metric shows how long various members were in PRIMARY and SECONDARY roles.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.max-heartbeat-time:

`Max Heartbeat Time <dashboard.mongodb-replset.html#max-heartbeat-time>`_
--------------------------------------------------------------------------------

This metric shows the heartbeat return times sent by the current member to other
members in the replica set.

Long heartbeat times can indicate network issues or that the server is too busy.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-replset.max-member-ping-time:

`Max Member Ping Time <dashboard.mongodb-replset.html#max-member-ping-time>`_
--------------------------------------------------------------------------------

This metric can show a correlation with the replication lag value.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard.mongodb-replset`

.. include:: ../.res/replace.txt

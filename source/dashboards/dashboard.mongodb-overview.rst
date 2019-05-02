.. _dashboard.mongodb-overview:

|mongodb| Overview
================================================================================

This dashboard provides basic information about |mongodb| instances.

.. _dashboard.mongodb-overview.command-operations:

`Command Operations <dashboard.mongodb-overview.html#command-operations>`_
--------------------------------------------------------------------------------

Shows how many times a command is executed per second on average during the
selected interval.

Look for peaks and drops and correlate them with other graphs.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.connections:

`Connections <dashboard.mongodb-overview.html#connections>`_
--------------------------------------------------------------------------------

Keep in mind the hard limit on the maximum number of connections set by your
distribution.

Anything over 5,000 should be a concern, because the application may not close
connections correctly.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.cursors:

`Cursors <dashboard.mongodb-overview.html#cursors>`_
--------------------------------------------------------------------------------

Helps identify why connections are increasing.  Shows active cursors compared to
cursors being automatically killed after 10 minutes due to an application not
closing the connection.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.document-operations:

`Document Operations <dashboard.mongodb-overview.html#document-operations>`_
--------------------------------------------------------------------------------

When used in combination with **Command Operations**, this graph can help
identify *write aplification*.  For example, when one ``insert`` or ``update``
command actually inserts or updates hundreds, thousands, or even millions of
documents.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.queued-operations:

`Queued Operations <dashboard.mongodb-overview.html#queued-operations>`_
--------------------------------------------------------------------------------

Any number of queued operations for long periods of time is an indication of
possible issues.  Find the cause and fix it before requests get stuck in the
queue.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.getlasterror-write-time.operations:

`getLastError Write Time, getLastError Write Operations <dashboard.mongodb-overview.html#getlasterror-write-time.operations>`_
-------------------------------------------------------------------------------------------------------------------------------

This is useful for write-heavy workloads to understand how long it takes to
verify writes and how many concurrent writes are occurring.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.asserts:

`Asserts <dashboard.mongodb-overview.html#asserts>`_
--------------------------------------------------------------------------------

Asserts are not important by themselves, but you can correlate spikes with other
graphs.

|view-all-metrics| |this-dashboard|

.. _dashboard.mongodb-overview.memory-faults:

`Memory Faults <dashboard.mongodb-overview.html#memory-faults>`_
--------------------------------------------------------------------------------

Memory faults indicate that requests are processed from disk either because an
index is missing or there is not enough memory for the data set.  Consider
increasing memory or sharding out.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard.mongodb-overview`

.. include:: ../.res/replace.txt

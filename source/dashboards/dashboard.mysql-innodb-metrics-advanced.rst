.. _dashboard.mysql-innodb-metrics-advanced:

|mysql| |innodb| Metrics (Advanced) Dashboard
================================================================================

The |mysql| |innodb| Metrics (Advanced) dashboard contains metrics that provide
detailed information about the performance of the |innodb| storage engine on the
selected |mysql| host. This dashboard contains the following metrics:

.. important::

   If you do not see any metric, try running the following command in the
   |mysql| client:

   .. include:: ../.res/code/set.global.innodb-monitor-enable.txt

.. rubric:: Metrics of |this-dashboard|
   
.. contents::
   :local:

.. _dashboard.mysql-innodb-metrics-advanced.change-buffer-performance:

Change Buffer Performance
--------------------------------------------------------------------------------

This metric shows the activity on the |innodb| change buffer.  The |innodb|
change buffer records updates to non-unique secondary keys when the destination
page is not in the buffer pool.  The updates are applied when the page is loaded
in the buffer pool, prior to its use by a query.  The merge ratio is the number
of insert buffer changes done per page, the higher the ratio the better is the
efficiency of the change buffer.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-log-buffer-performance:

InnoDB Log Buffer Performance
--------------------------------------------------------------------------------

The |innodb| Log Buffer Performance graph shows the efficiency of the |innodb|
log buffer.  The |innodb| log buffer size is defined by the
|opt.innodb-log-buffer-size| parameter and illustrated on the graph by the
*Size* graph.  *Used* is the amount of the buffer space that is used.  If the
*Used* graph is too high and gets close to *Size*, additional log writes will be
required.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-page-splits:

|innodb| Page Splits
--------------------------------------------------------------------------------

The |innodb| Page Splits graph shows the |innodb| page maintenance activity
related to splitting and merging pages.  When an |innodb| page, other than the
top most leaf page, has too much data to accept a row update or a row insert, it
has to be split in two.  Similarly, if an |innodb| page, after a row update or
delete operation, ends up being less than half full, an attempt is made to merge
the page with a neighbor page. If the resulting page size is larger than the
|innodb| page size, the operation fails.  If your workload causes a large number
of page splits, try lowering the innodb_fill_factor variable (5.7+).

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-page-reorgs:

|innodb| Page Reorgs
--------------------------------------------------------------------------------

The |innodb| Page Reorgs graph shows information about the page reorganization
operations.  When a page receives an update or an insert that affect the offset
of other rows in the page, a reorganization is needed.  If the reorganization
process finds out there is not enough room in the page, the page will be
split. Page reorganization can only fail for compressed pages.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-purge-performance:

|innodb| Purge Performance
--------------------------------------------------------------------------------

The |innodb| Purge Performance graph shows metrics about the page purging
process.  The purge process removed the undo entries from the history list and
cleanup the pages of the old versions of modified rows and effectively remove
deleted rows.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-locking:

|innodb| Locking
--------------------------------------------------------------------------------

The |innodb| Locking graph shows the row level lock activity inside |innodb|. 

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-main-thread-utilization:

|innodb| Main Thread Utilization
--------------------------------------------------------------------------------

The |innodb| Main Thread Utilization graph shows the portion of time the
|innodb| main thread spent at various task.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-transactions-information:

|innodb| Transactions Information
--------------------------------------------------------------------------------

The |innodb| Transactions Information graph shows details about the recent
transactions.  Transaction IDs Assigned represents the total number of
transactions initiated by |innodb|.  RW Transaction Commits are the number of
transactions not read-only. Insert-Update Transactions Commits are transactions
on the Undo entries.  Non Locking RO Transaction Commits are transactions commit
from select statement in auto-commit mode or transactions explicitly started
with "start transaction read only".

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-undo-space-usage:

|innodb| Undo Space Usage
--------------------------------------------------------------------------------

The |innodb| Undo Space Usage graph shows the amount of space used by the Undo
segment.  If the amount of space grows too much, look for long running
transactions holding read views opened in the |innodb| status.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-activity:

|innodb| Activity
--------------------------------------------------------------------------------

The |innodb| Acitivity graph shows a measure of the activity of the |innodb|
threads.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-contention-os-waits:

|innodb| Contention - OS Waits
--------------------------------------------------------------------------------

The |innodb| Contention - OS Waits graph shows the number of time an OS wait
operation was required while waiting to get the lock.  This happens once the
spin rounds are exhausted.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-contention-spin-rounds:

|innodb| Contention - Spin Rounds
--------------------------------------------------------------------------------

The |innodb| Contention - Spin Rounds metric shows the number of spin rounds
executed in order to get a lock.  A spin round is a fast retry to get the lock
in a loop.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-group-commit-batch-size:

|innodb| Group Commit Batch Size
--------------------------------------------------------------------------------

The |innodb| Group Commit Batch Size metric shows how many bytes were written to
the |innodb| log files per attempt to write.  If many threads are committing at
the same time, one of them will write the log entries of all the waiting threads
and flush the file.  Such process reduces the number of disk operations needed
and enlarge the batch size.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-purge-throttling:

|innodb| Purge Throttling
--------------------------------------------------------------------------------

The |innodb| Purge Throttling graph shows the evolution of the purge lag and the
max purge lag currently set.  Under heavy write load, the purge operation may
start to lag behind and when the max purge lag is reached, a delay, proportional
to the value defined by innodb_max_purge_lag_delay (in microseconds) is added to
all update, insert and delete statements.  This helps prevents flushing stalls.

https://dev.mysql.com/doc/refman/5.6/en/innodb-parameters.html#sysvar_innodb_max_purge_lag

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-ahi-usage:

|innodb| AHI Usage
--------------------------------------------------------------------------------

The |innodb| AHI Usage graph shows the search operations on the |innodb|
adaptive hash index and its efficiency.  The adaptive hash index is a search
hash designed to speed access to |innodb| pages in memory.  If the Hit Ratio is
small, the working data set is larger than the buffer pool, the AHI should
likely be disabled.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-ahi-maintenance:

|innodb| AHI Maintenance
--------------------------------------------------------------------------------

The |innodb| AHI Maintenance graph shows the maintenance operation of the
|innodb| adaptive hash index.  The adaptive hash index is a search hash to speed
access to |innodb| pages in memory. A constant high number of rows/pages added
and removed can be an indication of an ineffective AHI.

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-online-ddl:

|innodb| Online DDL
--------------------------------------------------------------------------------

The |innodb| Online DDL graph shows the state of the online DDL (alter table)
operations in |innodb|.  The progress metric is estimate of the percentage of
the rows processed by the online DDL.

.. note::

   Currently available only on |mariadb| Server

|view-all-metrics|
|this-dashboard|

.. _dashboard.mysql-innodb-metrics-advanced.innodb-defragmentation:

|innodb| Defragmentation
--------------------------------------------------------------------------------

The |innodb| Defragmentation graph shows the status information related to the
|innodb| online defragmentation feature of |mariadb| for the optimize table
command.  To enable this feature, the variable innodb-defragment must be set to
**1** in the configuration file.

|view-all-metrics|
|this-dashboard|

.. note::

   Currently available only on |mariadb| Server.

.. |this-dashboard| replace:: :ref:`dashboard.mysql-innodb-metrics-advanced`

.. include:: ../.res/replace.txt

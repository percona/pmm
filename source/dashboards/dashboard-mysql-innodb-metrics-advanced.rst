.. _dashboard-mysql-innodb-metrics-advanced:

###############################
MySQL InnoDB Metrics (Advanced)
###############################

The MySQL InnoDB Metrics (Advanced) dashboard contains metrics that provide
detailed information about the performance of the InnoDB storage engine on the
selected MySQL host. This dashboard contains the following metrics:

.. important::

   If you do not see any metric, try running the following command in the
   MySQL client:

   .. code-block:: mysql

      mysql > SET GLOBAL innodb_monitor_enable=all;


.. _dashboard-mysql-innodb-metrics-advanced.change-buffer-performance:

*************************
Change Buffer Performance
*************************

This metric shows the activity on the InnoDB change buffer.  The InnoDB
change buffer records updates to non-unique secondary keys when the destination
page is not in the buffer pool.  The updates are applied when the page is loaded
in the buffer pool, prior to its use by a query.  The merge ratio is the number
of insert buffer changes done per page, the higher the ratio the better is the
efficiency of the change buffer.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-log-buffer-performance:

*****************************
InnoDB Log Buffer Performance
*****************************

The InnoDB Log Buffer Performance graph shows the efficiency of the InnoDB
log buffer.  The InnoDB log buffer size is defined by the
``innodb_log_buffer_size`` parameter and illustrated on the graph by the
*Size* graph.  *Used* is the amount of the buffer space that is used.  If the
*Used* graph is too high and gets close to *Size*, additional log writes will be
required.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-page-splits:

******************
InnoDB Page Splits
******************

The InnoDB Page Splits graph shows the InnoDB page maintenance activity
related to splitting and merging pages.  When an InnoDB page, other than the
top most leaf page, has too much data to accept a row update or a row insert, it
has to be split in two.  Similarly, if an InnoDB page, after a row update or
delete operation, ends up being less than half full, an attempt is made to merge
the page with a neighbor page. If the resulting page size is larger than the
InnoDB page size, the operation fails.  If your workload causes a large number
of page splits, try lowering the ``innodb_fill_factor variable`` (5.7+).

.. _dashboard-mysql-innodb-metrics-advanced.innodb-page-reorgs:

******************
InnoDB Page Reorgs
******************

The InnoDB Page Reorgs graph shows information about the page reorganization
operations.  When a page receives an update or an insert that affect the offset
of other rows in the page, a reorganization is needed.  If the reorganization
process finds out there is not enough room in the page, the page will be
split. Page reorganization can only fail for compressed pages.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-purge-performance:

************************
InnoDB Purge Performance
************************

The InnoDB Purge Performance graph shows metrics about the page purging
process.  The purge process removed the undo entries from the history list and
cleanup the pages of the old versions of modified rows and effectively remove
deleted rows.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-locking:

**************
InnoDB Locking
**************

The InnoDB Locking graph shows the row level lock activity inside InnoDB.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-main-thread-utilization:

******************************
InnoDB Main Thread Utilization
******************************

The InnoDB Main Thread Utilization graph shows the portion of time the
InnoDB main thread spent at various task.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-transactions-information:

*******************************
InnoDB Transactions Information
*******************************

The InnoDB Transactions Information graph shows details about the recent
transactions.  Transaction IDs Assigned represents the total number of
transactions initiated by InnoDB.  RW Transaction Commits are the number of
transactions not read-only. Insert-Update Transactions Commits are transactions
on the Undo entries.  Non Locking RO Transaction Commits are transactions commit
from select statement in auto-commit mode or transactions explicitly started
with "start transaction read only".

.. _dashboard-mysql-innodb-metrics-advanced.innodb-undo-space-usage:

***********************
InnoDB Undo Space Usage
***********************

The InnoDB Undo Space Usage graph shows the amount of space used by the Undo
segment.  If the amount of space grows too much, look for long running
transactions holding read views opened in the InnoDB status.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-activity:

***************
InnoDB Activity
***************

The InnoDB Acitivity graph shows a measure of the activity of the InnoDB
threads.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-contention-os-waits:

****************************
InnoDB Contention - OS Waits
****************************

The InnoDB Contention - OS Waits graph shows the number of time an OS wait
operation was required while waiting to get the lock.  This happens once the
spin rounds are exhausted.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-contention-spin-rounds:

*******************************
InnoDB Contention - Spin Rounds
*******************************

The InnoDB Contention - Spin Rounds metric shows the number of spin rounds
executed in order to get a lock.  A spin round is a fast retry to get the lock
in a loop.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-group-commit-batch-size:

******************************
InnoDB Group Commit Batch Size
******************************

The InnoDB Group Commit Batch Size metric shows how many bytes were written to
the InnoDB log files per attempt to write.  If many threads are committing at
the same time, one of them will write the log entries of all the waiting threads
and flush the file.  Such process reduces the number of disk operations needed
and enlarge the batch size.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-purge-throttling:

***********************
InnoDB Purge Throttling
***********************

The InnoDB Purge Throttling graph shows the evolution of the purge lag and the
max purge lag currently set.  Under heavy write load, the purge operation may
start to lag behind and when the max purge lag is reached, a delay, proportional
to the value defined by ``innodb_max_purge_lag_delay`` (in microseconds) is added to
all update, insert and delete statements.  This helps prevents flushing stalls.

.. seealso::

   https://dev.mysql.com/doc/refman/5.6/en/innodb-parameters.html#sysvar_innodb_max_purge_lag

.. _dashboard-mysql-innodb-metrics-advanced.innodb-ahi-usage:

****************
InnoDB AHI Usage
****************

The InnoDB AHI Usage graph shows the search operations on the InnoDB
adaptive hash index and its efficiency.  The adaptive hash index is a search
hash designed to speed access to InnoDB pages in memory.  If the Hit Ratio is
small, the working data set is larger than the buffer pool, the AHI should
likely be disabled.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-ahi-maintenance:

**********************
InnoDB AHI Maintenance
**********************

The InnoDB AHI Maintenance graph shows the maintenance operation of the
InnoDB adaptive hash index.  The adaptive hash index is a search hash to speed
access to InnoDB pages in memory. A constant high number of rows/pages added
and removed can be an indication of an ineffective AHI.

.. _dashboard-mysql-innodb-metrics-advanced.innodb-online-ddl:

*****************
InnoDB Online DDL
*****************

The InnoDB Online DDL graph shows the state of the online DDL (alter table)
operations in InnoDB.  The progress metric is estimate of the percentage of
the rows processed by the online DDL.

.. note::

   Currently available only on MariaDB Server

.. _dashboard-mysql-innodb-metrics-advanced.innodb-defragmentation:

**********************
InnoDB Defragmentation
**********************

The InnoDB Defragmentation graph shows the status information related to the
InnoDB online defragmentation feature of MariaDB for the optimize table
command.  To enable this feature, the variable ``innodb-defragment`` must be set to
``1`` in the configuration file.

.. note::

   Currently available only on MariaDB Server.

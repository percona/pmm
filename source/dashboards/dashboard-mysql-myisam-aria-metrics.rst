.. _dashboard-mysql-myisam-aria-metrics:

MySQL MyISAM Aria Metrics Dashboard
================================================================================

The MySQL MyISAM Aria Metrics dashboard describes the specific features
of MariaDB MySQL server: `Aria storage engine <https://mariadb.com/kb/en/the-mariadb-library/aria-storage-engine/>`_, `Online DDL (online alter table) <https://mariadb.com/kb/en/the-mariadb-library/alter-table/>`_,
and `InnoDB defragmentation patch <https://mariadb.com/kb/en/the-mariadb-library/defragmenting-innodb-tablespaces/>`_. This dashboard contains the following metrics:

.. contents::
   :local:

.. _dashboard-mysql-myisam-aria-metrics.aria-storage-engine:

Aria Storage Engine
--------------------------------------------------------------------------------

Aria storage is specific for MariaDB Server. Aria has most of the same
variables that MyISAM has, but with an Aria prefix. If you use Aria
instead of MyISAM, then you should make ``key_buffer_size`` smaller and
aria-pagecache-buffer-size bigger.

.. _dashboard-mysql-myisam-aria-metrics.aria-pagecache-reads-writes:

Aria Pagecache Reads/Writes
--------------------------------------------------------------------------------

This graph is similar to InnoDB buffer pool reads and
writes. ``aria-pagecache-buffer-size`` is the main cache for aria storage
engine. If you see high reads and writes (physical IO), i.e. reads is close to
read requests or writes are close to write requests you may need to increase the
``aria-pagecache-buffer-size`` (you may need to decrease other buffers:
``key_buffer_size``, ``innodb_buffer_pool_size`` etc)

.. _dashboard-mysql-myisam-aria-metrics.aria-pagecache-blocks:

Aria Pagecache Blocks
--------------------------------------------------------------------------------

This graphs shows the utilization for the aria pagecache.  This is similar to
InnoDB buffer pool graph. If you see all blocks are used you may consider
increasing ``aria-pagecache-buffer-size`` (you may need to decrease other
buffers: ``key_buffer_size``, ``innodb_buffer_pool_size`` etc)

.. _dashboard-mysql-myisam-aria-metrics.aria-transactions-log-syncs:

Aria Transactions Log Syncs
--------------------------------------------------------------------------------

This metric is similar to InnoDB log file syncs. If you see lots of log syncs
and want to relax the durability settings you can change (in seconds) from 30
(default) to a higher number. It is good to look at the disk IO dashboard as
well.

.. seealso::

   List of Aria system variables

      https://mariadb.com/kb/en/library/aria-system-variables/




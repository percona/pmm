
.. _dashboard-mysql-myrocks-metrics:

|dbd.mysql-myrocks-metrics| Dashboard
================================================================================

The MyRocks_ storage engine developed by |facebook| based on the |rocksdb|
storage engine is applicable to systems which primarily interact with the
database by writing data to it rather than reading from it. |rocksdb| also
features a good level of compression, higher than that of the |innodb| storage
engine, which makes it especially valuable when optimizing the usage of hard
drives.

|pmm| collects statistics on the |myrocks| storage engine for |mysql| in the
|metrics-monitor| information for this dashboard comes from the
|inf-schema| tables.

.. figure:: ../.res/graphics/png/metrics-monitor.mysql-myrocks-metrics.1.png
	    
   The |mysql| |myrocks| metrics dashboard

.. seealso::

   Information schema
      https://github.com/facebook/mysql-5.6/wiki/MyRocks-Information-Schema

.. rubric:: Metrics
	    
.. hlist::
   :columns: 2

   - |myrocks| cache
   - |myrocks| cache data bytes R/W
   - |myrocks| cache index hit rate
   - |myrocks| cache index
   - |myrocks| cache filter hit rate
   - |myrocks| cache filter
   - |myrocks| cache data byltes inserted
   - |myrocks| bloom filter
   - |myrocks| memtable
   - |myrocks| memtable size
   - |myrocks| number of keys
   - |myrocks| cache L0/L1
   - |myrocks| number of DB ops
   - |myrocks| R/W
   - |myrocks| bytes read by iterations
   - |myrocks| write ops
   - |myrocks| WAL
   - |myrocks| number reseeks in iterations
   - |rocksdb| row operations
   - |myrocks| file operations
   - |rocksdb| stalls
   - |rocksdb| stops/slowdowns

.. _myrocks: http://myrocks.io

.. include:: ../.res/replace.txt


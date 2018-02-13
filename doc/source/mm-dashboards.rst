.. _mm-dashboards:

================================================================================
Metrics Monitor Dashboards
================================================================================

This section contains a reference of dashboards available in |metrics-monitor|.

.. contents::
   :local:

MySQL Overview
================================================================================

This dashboard provides basic information about MySQL hosts.

.. include:: .res/table/list-table.org
   :start-after: +dashboard.mysql-overview+
   :end-before: #+end-block

.. seealso::

   Configuring |mysql| for |pmm|
      :ref:`pmm/mysql/conf/dashboard`
   |mysql| Documentation: |innodb| buffer pool
      https://dev.mysql.com/doc/refman/5.7/en/innodb-buffer-pool.html
   |mysql| Server System Variables: key_buffer_size
      https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_key_buffer_size
   |percona-server| Documentation: Running |tokudb| in Production
      https://www.percona.com/doc/percona-server/LATEST/tokudb/tokudb_quickstart.html#considerations-to-run-tokudb-in-production
   Blog post: Adaptive Hash Index in |innodb|
      https://www.percona.com/blog/2016/04/12/is-adaptive-hash-index-in-innodb-right-for-my-workload/

MySQL Query Response Time
================================================================================

This dashboard provides information about query response time distribution. 

.. include:: .res/table/list-table.org
   :start-after: +dashboard.mysql-query-response-time+
   :end-before: #+end-block

.. seealso::
   
   Configuring |mysql| for |pmm|
      :ref:`pmm/mysql/conf/dashboard/mysql-query-response-time`


MongoDB Overview
================================================================================

This dashboard provides basic information about MongoDB instances.

.. include:: .res/table/list-table.org
   :start-after: +dashboard.mongodb-overview+
   :end-before: #+end-block

MongoDB ReplSet
================================================================================

This dashboard provides information about replica sets and their members.

.. include:: .res/table/list-table.org
   :start-after: +dashboard.mongodb-replset+
   :end-before: #+end-block


Cross Server Graphs
================================================================================

.. include:: .res/table/list-table.org
   :start-after: +dashboard.cross-server-graphs+
   :end-before: #+end-block


.. include:: .res/replace/program.txt
.. include:: .res/replace/name.txt

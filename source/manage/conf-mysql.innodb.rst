`MySQL InnoDB Metrics <pmm.conf-mysql.mysql-innodb.metrics>`_
================================================================================

Collecting metrics and statistics for graphs increases overhead.  You can keep
collecting and graphing low-overhead metrics all the time, and enable
high-overhead metrics only when troubleshooting problems.

InnoDB metrics provide detailed insight about |innodb| operation.  Although you
can select to capture only specific counters, their overhead is low even when
they all are enabled all the time. To enable all |innodb| metrics, set the
global variable |opt.innodb-monitor-enable| to ``all``:

.. code-block:: sql

   mysql> SET GLOBAL innodb_monitor_enable=all

.. seealso::

   |mysql| Documentation: |opt.innodb-monitor-enable| variable
      https://dev.mysql.com/doc/refman/5.7/en/innodb-parameters.html#sysvar_innodb_monitor_enable

.. include:: .res/replace.txt

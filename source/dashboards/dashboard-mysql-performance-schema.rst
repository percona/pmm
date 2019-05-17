
.. _dashboard-mysql-performance-schema:

|mysql| |perf-schema|  Dashboard
================================================================================

The |mysql| |perf-schema| dashboard helps determine the efficiency of
communicating with |performance-schema|. This dashboard contains the following
metrics:

.. hlist::
   :columns: 2

   - |perf-schema| file IO (events)
   - |perf-schema| file IO (load)
   - |perf-schema| file IO (Bytes)
   - |perf-schema| waits (events)
   - |perf-schema| waits (load)
   - Index access operations (load)
   - Table access operations (load)
   - |perf-schema| SQL and external locks (events)
   - |perf-schema| SQL and external locks (seconds)

.. seealso::

   |mysql| Documentation: |performance-schema|

      https://dev.mysql.com/doc/refman/5.7/en/performance-schema.html

.. include:: ../.res/replace.txt

:orphan: true

.. _pmm/exporter-option.proxysql:

================================================================================
|proxysql| Server Exporter (proxysql_exporter)
================================================================================

The following options may be passed to the |opt.proxysql-metrics| monitoring
service as additional options. For more information about this exporter see its
|github| repository: https://github.com/percona/proxysql_exporter.

.. seealso::

   Adding monitoring services
      :ref:`pmm-admin.add`
   Passing options to a monitoring service
      :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`
   All exporter options
      :ref:`pmm/list.exporter-option`

Collector Options
================================================================================

.. include:: .res/table/table.org
   :start-after: +proxysql-exporter.collector-flag+
   :end-before: #+end-table

General Options
================================================================================

.. include:: .res/table/table.org
   :start-after: +proxysql-exporter.general-flag+
   :end-before: #+end-table

.. include:: .res/replace/option.txt
.. include:: .res/replace/name.txt


   

.. _pmm.exporter.node:

Node Exporter (node_exporter)
================================================================================

The following options may be passed to the |opt.linux-metrics| monitoring
service as additional options. For more information about this exporter see its
|github| repository: https://github.com/percona/node_exporter.

.. seealso::

   Adding monitoring services
      :ref:`pmm-admin.add`
   Passing options to a monitoring service
      :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`
   All exporter options
      :ref:`pmm.list.exporter`

.. _pmm.exporter.node.collector-option:

.. rubric:: Collector options

.. include:: ../.res/table/node-exporter.flag.txt

.. important::

   .. versionadded:: 1.13.0

   |pmm| shows NUMA related metrics on the |dbd.advanced-data-exploration| and
   |dbd.overview-numa-metrics| dashboards. To enable this feature, the
   |opt.meminfo-numa| option is enabled automatically when you install |pmm|.


.. admonition:: Relatedion information

   Setting collector options
      https://github.com/prometheus/node_exporter#collectors

.. include:: ../.res/replace.txt

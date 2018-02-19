:orphan: true

.. _pmm/exporter-option.mysqld:

================================================================================
|mysql| Server Exporter (mysqld_exporter)
================================================================================

|mysqld_exporter| is the |prometheus| exporter for |mysql|
metrics. This exporter has three resolutions to group the metrics:

- metrics-lr (metrics with a low resolution)
- metrics-mr (metrics with a medium resolution)
- metrics-hr (metrics with a high resolution)

For example, *metrics-hr* contains very frequently changing values, such as |mysql-global-status-commands-total|.
On the other hand, *metrics-lr* contains infrequently changing values such as |mysql-global-variables-autocommit|.

The following options may be passed to the |opt.mysql-metrics| monitoring
service as additional options.

.. seealso::

   Adding monitoring services
      :ref:`pmm-admin.add`
   Passing options to a monitoring service
      :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`
   All exporter options
      :ref:`pmm/list.exporter-option`

Collector options
================================================================================

.. include:: .res/table/table.org
   :start-after: +mysqld-exporter.collector-flag+
   :end-before: #+end-table

General options
================================================================================

.. include:: .res/table/table.org
   :start-after: +mysqld-exporter.collector-flag+
   :end-before: #+end-table

.. include:: .res/replace/option.txt
.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt

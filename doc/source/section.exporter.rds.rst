:orphan: true

.. _pmm/exporter/rds:

================================================================================
|amazon-rds| Exporter (rds_exporter)
================================================================================

The |amazon-rds| exporter makes the |amazon-cloudwatch| metrics available to
|pmm|. |pmm| uses this exporter to obtain metrics from any |amazon-rds| node
that you choose to monitor.

.. seealso::

   Repository on Github
      https://github.com/percona/rds_exporter
   Adding monitoring services
      :ref:`pmm-admin.add`
   Passing options to a monitoring service
      :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`
   All exporter options
      :ref:`pmm/list.exporter`

.. _pmm/exporter/rds/metrics:

Metrics
================================================================================

The |amazon-rds| exporter has two types of metrics: basic and advanced. To be
able to use advanced metrics, make sure to set the
|gui.enable-enhanced-monitoring| option in the settings of your |amazon-rds| DB
instance.

.. figure:: .res/graphics/png/amazon-rds.modify-db-instance.2.png

   Set the |gui.enable-enhanced-monitoring| option in the settings of your
   |amazon-rds| DB instance.

.. seealso::

   |amazon-rds| Documentation: 
      - `Modifying an Amazon RDS DB Instance <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html>`_
      - `More information about enhanced monitoring <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.OS.html>`_
      - `Metrics and Dimensions <https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/rds-metricscollected.html>`_

.. include:: .res/table/list-table.org
   :start-after: +exporter.rds.metrics+
   :end-before: #+end-block
   
.. include:: .res/replace/name.txt

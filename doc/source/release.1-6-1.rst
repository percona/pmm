:orphan: true

.. _pmm/release/1-6-1:

Percona Monitoring and Management |release|
********************************************************************************

:Date: January 25, 2018

For more information about this release, see the `release announcement`_.

Issues in this release
================================================================================

|tip.bug-fix-release|
|pmm.name|
|prev-release|.

.. rubric:: Bug fixes

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1660`
     - |qan| for |mongodb| would not display data when authentication was enabled.
   * - :pmmbug:`1822`
     - In |metrics-monitor|, some tag names were incorrect in dashboards.
   * - :pmmbug:`1832`
     - After upgrading to 1.5.2, it was not possible to disable an |rds| instance on the :guilabel:`Add RDS instance` dashboard of |metrics-monitor|.
   * - :pmmbug:`1907`
     - In |metrics-monitor|, the tooltip of the *Engine Uptime* metric referred to an incorrect unit of measure.
   * - :pmmbug:`1964`
     - The same value of the *Exporter Uptime* metric could appear in different colors in different contexts.
   * - :pmmbug:`1965`
     - In the *Prometheus Exporters Overview* dashboard of |metrics-monitor|, the drill down links of some metrics could direct to a wrong host.

.. seealso::

   All releases
      :ref:`pmm/release/list`

   To release |prev-release|
      :ref:`pmm/release/1-6-0`

.. _`release announcement`: https://www.percona.com/blog/2018/01/25/percona-monitoring-and-management-pmm-1-6-1-is-now-available/

.. |release|      replace:: 1.6.1
.. |prev-release| replace:: 1.6.0

.. include:: .res/replace.txt

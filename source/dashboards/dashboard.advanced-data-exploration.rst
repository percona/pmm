.. _dashboard.advanced-data-exploration:

|dbd.advanced-data-exploration| Dashboard
================================================================================

The *Advanced Data Exploration* dashboard provides detailed information about
the progress of a single |prometheus| metric across one or more hosts.

.. admonition:: Added NUMA related metrics

   .. versionadded:: 1.13.0

   The |dbd.advanced-data-exploration| dashboard supports
   metrics related to NUMA. The names of all these metrics start with
   *node_memory_numa*.

   .. image:: ../.res/graphics/png/metrics-monitor.advanced-data-exploration.node-memory-numa.png

..  contents::
    :local:

.. _dashboard.advanced-data-exploration.metric-value.view-as-gauge:

:ref:`View actual metric values (Gauge) <dashboard.advanced-data-exploration.metric-value.view-as-gauge>`
---------------------------------------------------------------------------------------------------------

In this section, the values of the selected metric may increase or decrease over
time (similar to temperature or memory usage).

.. _dashboard.advanced-data-exploration.metric-value.view-as-counter:

:ref:`View actual metric values (Counters) <dashboard.advanced-data-exploration.metric-value.view-as-counter>`
--------------------------------------------------------------------------------------------------------------

In this section, the values of the selected metric are accummulated over time
(useful to count the number of served requests, for example).

.. _dashboard.advanced-data-exploration.metric-data-table:

:ref:`View actual metric values (Counters) <dashboard.advanced-data-exploration.metric-data-table>`
---------------------------------------------------------------------------------------------------

This section presents the values of the selected metric in the tabular form.

.. seealso::

   |prometheus| Documentation: Metric types
       https://prometheus.io/docs/concepts/metric_types/

.. include:: ../.res/replace.txt

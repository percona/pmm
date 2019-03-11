--------------------------------------------------------------------------------
Navigating across Dashboards
--------------------------------------------------------------------------------

Beside the |gui.dashboard-dropdown| button you can also Navigate across 
Dashboards with the navigation menu which groups dashboards by
application. Click the required group and then select the dashboard
that matches your choice.

.. _table.pmm.metrics-monitor.navigation-menu-group:

=============  ==============================================================
Group          Dashboards for monitoring ...
=============  ==============================================================   
|qan.name|     |qan| component (see :ref:`pmm.qan`)
OS             The operating system status
|mysql|        |mysql| and |amazon-aurora|
|mongodb|      State of |mongodb| hosts
HA             High availability
Cloud          |amazon-rds| and |amazon-aurora|
Insight        Summary, cross-server and |prometheus|
|pmm|          Server settings
=============  ==============================================================

.. figure:: .res/graphics/png/metrics-monitor.menu.png

   |mysql| group selected in the navigation menu


.. _pmm.metrics-monitor.metric.zooming-in:

`Zooming in on a single metric <pmm.metrics-monitor.metric.zooming-in>`_
================================================================================
     
On dashboards with multiple metrics, it is hard to see how the value of a single
metric changes over time. Use the context menu to zoom in on the selected metric
so that it temporarily occupies the whole dashboard space.

Click the title of the metric that you are interested in and select the
|gui.view| option from the context menu that opens.

.. figure:: .res/graphics/png/metrics-monitor.metric-context-menu.1.png

   The context menu of a metric

The selected metric opens to occupy the whole dashboard space. You may now set
another time range using the time and date range selector at the top of the
|metrics-monitor| page and analyze the metric data further.

.. figure:: .res/graphics/png/metrics-monitor.cross-server-graphs.load-average.1.png

To return to the dashboard, click the |gui.back-to-dashboard| button next to the time range selector.

.. figure:: .res/graphics/png/metrics-monitor.time-range-selector.1.png

   The |gui.back-to-dashboard| button returns to the dashboard; this button
   appears when you are zooming in on one metric.

Navigation menu allows you to navigate between dashboards while maintaining the
same host under observation and/or the same selected time range, so that for
example you can start on *MySQL Overview* looking at host serverA, switch to
MySQL InnoDB Advanced dashboard and continue looking at serverA, thus saving you
a few clicks in the interface.

.. include:: .res/replace.txt

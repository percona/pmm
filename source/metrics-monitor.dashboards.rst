--------------------------------------------------------------------------------
Understanding Dashboards
--------------------------------------------------------------------------------

The |metrics-monitor| tool provides a |metrics-monitor.what-is|. Time-based
graphs are separated into dashboards by themes: some are related to |mysql| or
|mongodb|, others provide general system metrics.

.. _pmm.metrics-monitor.dashboard.opening:

`Opening a Dashboard <metrics-monitor.dashboards.html#pmm-metrics-monitor-dashboard-opening>`_
==============================================================================================

The default |pmm| installation provides more than thirty dashboards. To make it
easier to reach a specific dashboard, the system offers two tools. The
|gui.dashboard-dropdown| is a button in the header of any |pmm| page. It lists
all dashboards, organized into folders. Right sub-panel allows to rearrange
things, creating new folders and dragging dashboards into them. Also a text box
on the top allows to search the required dashboard by typing.

.. figure:: .res/graphics/png/metrics-monitor.dashboard-dropdown.png

   With |gui.dashboard-dropdown|, search the alphabetical list for any
   dashboard.

.. _pmm.metrics-monitor.graph-description:

`Viewing More Information about a Graph <metrics-monitor.dashboards.html#pmm-metrics-monitor-graph-description>`_
==================================================================================================================

Each graph has a descriptions to display more information about the monitored
data without cluttering the interface.

These are on-demand descriptions in the tooltip format that you can find by
hovering the mouse pointer over the |gui.more-information| icon at the top left
corner of a graph. When you move the mouse pointer away from the |gui.more-inf|
button the description disappears.

.. figure:: .res/graphics/png/metrics-monitor.description.1.png

   Graph descriptions provide more information about a graph without claiming
   any space in the interface.

.. seealso::

   More information about the time range selector
      :ref:`Selecting time or date range <pmm.qan.time-date-range.selecting>`

.. include:: .res/replace.txt

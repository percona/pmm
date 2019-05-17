.. _dashboard-trends:

Trends Dashboard
================================================================================

The *Trends* dashboard shows the essential statistics about the selected
host. It also includes the essential statistics of |mysql|, such as |mysql|
questions and |innodb| row reads and row changes.

.. note::

   The |mysql| statistics section is empty for hosts other than |mysql|.

.. seealso::

   |mysql| Documentation: 

      `Questions
      <https://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html#statvar_Questions>`_

.. rubric:: Metrics
	    
.. contents::
   :local:

.. _dashboard-trends.cpu-usage:

CPU Usage
--------------------------------------------------------------------------------

This metric shows the comparison of the percentage of the CPU usage for the
current selected range, the previous day and the previous week.
This graph is useful to demonstrate how the CPU usage has changed over time by
visually overlaying time periods.
 
|view-all-metrics| |this-dashboard|

.. _dashboard-trends.io-read-activity:

I/O Read Activity
--------------------------------------------------------------------------------

This metric shows the comparison of I/O Read Activity in terms of bytes read for
the current selected range versus the previous day and the previous week for the
same time range. This graph is useful to demonstrate how I/O Read Activity has
changed over time by visually overlaying time periods. 

|view-all-metrics| |this-dashboard| 

.. _dashboard-trends.io-write-activity:

I/O Write Activity
--------------------------------------------------------------------------------

Shows the comparison of I/O Write Activity in terms of byte written for the
current selected range versus the previous day and the previous week for the
same time range. This graph is useful to demonstrate how I/O Write Activity has
changed over time by visually overlaying time periods.

|view-all-metrics| |this-dashboard|

.. _dashboard-trends.mysql-questions:

|mysql| Questions
--------------------------------------------------------------------------------

This metric shows the comparison of the |mysql| Questions for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how |mysql| Questions has changed
over time by visually overlaying time periods.

|view-all-metrics| |this-dashboard|

.. _dashboard-trends.innodb-rows-read:

|innodb| Rows Read
--------------------------------------------------------------------------------

This metric shows the comparison of the |innodb| Rows Read for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how |innodb| Rows Read has changed
over time by visually overlaying time periods.

|view-all-metrics| |this-dashboard|

.. _dashboard-trends.innodb-rows-changed:

|innodb| Rows Changed
--------------------------------------------------------------------------------

This metric shows the comparison of |innodb| Rows Changed for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how the |innodb| Rows Changed has
fluctuated over time by visually overlaying time periods.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard-trends`

.. include:: ../.res/replace.txt

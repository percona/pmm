.. _dashboard.trends:

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

.. _dashboard.trends.cpu-usage:

CPU Usage
--------------------------------------------------------------------------------

This metric shows the comparison of the percentage of the CPU usage for the
current selected range, the previous day and the previous week.

Use this metric to see how the CPU load increases or decreases when deploying a
new application or experiencing a different traffic on your site or to confirm
that your CPUs are working the same for your scheduled tasks.
 
|view-all-metrics| |this-dashboard|

.. _dashboard.trends.io-read-activity:

I/O Read Activity
--------------------------------------------------------------------------------

This metric shows the comparison of the I/O read activity in terms of byte read
for the currently selected range, the previous day, and the previous week.

Use this metric to see how the I/O read activity increases or decreases when
deploying a new application or experiencing a different traffic on your site or
to confirm that your I/O read activity is working the same for your scheduled
tasks.

|view-all-metrics| |this-dashboard| 

.. _dashboard.trends.io-write-activity:

I/O Write Activity
--------------------------------------------------------------------------------

Shows the comparison of the I/O write activity in terms of byte written for the
current selected range, the previous day, and the previous week.

Use this metric to see how the I/O write activity increases or decreases when
deploying a new application or experiencing a different traffic on your site or
to confirm that your I/O write activity is working the same for your scheduled
tasks.

|view-all-metrics| |this-dashboard|

.. _dashboard.trends.mysql-questions:

|mysql| Questions
--------------------------------------------------------------------------------

This metric shows the comparison of the |mysql| Questions for the current
selected range, the previous day, and the previous week.

Use this metric to see how the |mysql| questions increases or decreases when
deploying a new application or experiencing a different traffic on your site or
to confirm that your |mysql| questions are working the same for your scheduled
tasks.

|view-all-metrics| |this-dashboard|

.. _dashboard.trends.innodb-rows-read:

|innodb| Rows Read
--------------------------------------------------------------------------------

This metric shows the comparison of the |innodb| rows read for the current
selected range, the previous day, and the previous week.

Use this metric to see how the |innodb| rows read increases or decreases when
deploying a new application or experiencing a different traffic on your site or
to confirm that your |innodb| rows read are the same for your scheduled tasks.

|view-all-metrics| |this-dashboard|

.. _dashboard.trends.innodb-rows-changed:

|innodb| Rows Changed
--------------------------------------------------------------------------------

This metric shows the comparison of the |innodb| Rows Change for the current
selected range, the previous day and the previous week.

Use this metric to see how the |innodb| rows changed increases or decreases when
deploying a new application or experiencing a different traffic on your site or
to confirm that your |innodb| rows changed are the same for your scheduled tasks.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard.trends`

.. include:: .res/replace/name.txt
.. include:: .res/replace/fragment.txt

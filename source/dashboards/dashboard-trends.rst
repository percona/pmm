:orphan:

.. _dashboard-trends:

######
Trends
######

The *Trends* dashboard shows the essential statistics about the selected
host. It also includes the essential statistics of MySQL, such as MySQL
questions and InnoDB row reads and row changes.

.. note::

   The MySQL statistics section is empty for hosts other than MySQL.

.. seealso::

   MySQL Documentation:

      `Questions
      <https://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html#statvar_Questions>`_

.. rubric:: Metrics

.. _dashboard-trends.cpu-usage:

*********
CPU Usage
*********

This metric shows the comparison of the percentage of the CPU usage for the
current selected range, the previous day and the previous week.
This graph is useful to demonstrate how the CPU usage has changed over time by
visually overlaying time periods.

.. _dashboard-trends.io-read-activity:

*****************
I/O Read Activity
*****************

This metric shows the comparison of I/O Read Activity in terms of bytes read for
the current selected range versus the previous day and the previous week for the
same time range. This graph is useful to demonstrate how I/O Read Activity has
changed over time by visually overlaying time periods.

.. _dashboard-trends.io-write-activity:

******************
I/O Write Activity
******************

Shows the comparison of I/O Write Activity in terms of byte written for the
current selected range versus the previous day and the previous week for the
same time range. This graph is useful to demonstrate how I/O Write Activity has
changed over time by visually overlaying time periods.

.. _dashboard-trends.mysql-questions:

***************
MySQL Questions
***************

This metric shows the comparison of the MySQL Questions for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how MySQL Questions has changed
over time by visually overlaying time periods.

.. _dashboard-trends.innodb-rows-read:

****************
InnoDB Rows Read
****************

This metric shows the comparison of the InnoDB Rows Read for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how InnoDB Rows Read has changed
over time by visually overlaying time periods.

.. _dashboard-trends.innodb-rows-changed:

*******************
InnoDB Rows Changed
*******************

This metric shows the comparison of InnoDB Rows Changed for the current
selected range versus the previous day and the previous week for the same time
range. This graph is useful to demonstrate how the InnoDB Rows Changed has
fluctuated over time by visually overlaying time periods.

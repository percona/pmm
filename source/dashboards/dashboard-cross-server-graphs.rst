.. _dashboard-cross-server-graphs:

Cross Server Graphs
================================================================================

.. contents::
   :local:

.. _dashboard-cross-server-graphs.load-average:

Load Average
--------------------------------------------------------------------------------

This metric is the average number of processes that are either in a runnable or
uninterruptable state.  A process in a runnable state is either using the CPU or
waiting to use the CPU.  A process in uninterruptable state is waiting for some
I/O access, e.g., waiting for disk.

This metric is best used for trends. If you notice the load average rising, it
may be due to inefficient queries. In that case, you may further analyze your
queries in :ref:`QAN <QAN>`.

|view-all-metrics| |this-dashboard|

.. seealso::

   Description of *load average* in the man page of the |uptime| command in Debian
      https://manpages.debian.org/stretch/procps/uptime.1.en.html

.. _dashboard-cross-server-graphs.mysql-queries:

MySQL Queries
--------------------------------------------------------------------------------

This metric is based on the queries reported by the MySQL command
|sql.show-status|. It shows the average number of statements executed by the
server. This variable includes statements executed within stored programs,
unlike the |opt.questions| variable. It does not count *COM_PING* or
*COM_STATISTICS* commands.

|view-all-metrics| |this-dashboard|

.. seealso::

   MySQL Server Status Variables: Queries
      https://dev.mysql.com/doc/refman/5.6/en/server-status-variables.html#statvar_Queries

.. _dashboard-cross-server-graphs.mysql-traffic:

MySQL Traffic
--------------------------------------------------------------------------------

This metric shows the network traffic used by the MySQL process.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard-cross-server-graphs`

.. include:: ../.res/replace.txt

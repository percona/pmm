.. _pmm.conf-mysql.user-statistics:

####################################
MySQL User Statistics (``userstat``)
####################################

User statistics is a feature of Percona Server and MariaDB.  It provides
information about user activity, individual table and index access.  In some
cases, collecting user statistics can lead to high overhead, so use this feature
sparingly.

To enable user statistics, set the ``userstat`` variable to ``1``.

.. seealso::

   Percona Server Documentation: ``userstat``
      https://www.percona.com/doc/percona-server/5.7/diagnostics/user_stats.html#userstat

   MySQL Documentation
      `Setting variables <https://dev.mysql.com/doc/refman/5.7/en/set-variable.html>`_

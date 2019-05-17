`MySQL User Statistics (userstat) <pmm.conf-mysql.user-statistics>`_
--------------------------------------------------------------------------------

User statistics is a feature of |percona-server| and |mariadb|.  It provides
information about user activity, individual table and index access.  In some
cases, collecting user statistics can lead to high overhead, so use this feature
sparingly.

To enable user statistics, set the |opt.userstat| variable to ``1``.

.. seealso::

   |percona-server| Documentation: |opt.userstat|
      https://www.percona.com/doc/percona-server/5.7/diagnostics/user_stats.html#userstat

   |mysql| Documentation
      `Setting variables <https://dev.mysql.com/doc/refman/5.7/en/set-variable.html>`_

.. include:: ../.res/replace.txt

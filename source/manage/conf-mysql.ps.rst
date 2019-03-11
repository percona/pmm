`Percona Server specific settings <pmm.conf-mysql.settings.ps>`_
==================================================================

Not all dashboards in |metrics-monitor| are available by default for all |mysql|
variants and configurations: |oracle|'s |mysql|, |percona-server|. or |mariadb|.
Some graphs require |percona-server|, and specialized plugins, or additional
configuration.

`MySQL User Statistics <pmm.conf-mysql.user-statistics>`_
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

`Percona Server Query Response Time Distribution <pmm.conf-mysql.query-response-time>`_
-------------------------------------------------------------------------------------------

Query response time distribution is a feature available in |percona-server|.  It
provides information about changes in query response time for different groups
of queries, often allowing to spot performance problems before they lead to
serious issues.

To enable collection of query response time:

1. Install the |query-response-time| plugins:

   .. code-block:: sql

      mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_AUDIT SONAME 'query_response_time.so';
      mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME SONAME 'query_response_time.so';
      mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_READ SONAME 'query_response_time.so';
      mysql> INSTALL PLUGIN QUERY_RESPONSE_TIME_WRITE SONAME 'query_response_time.so';

#. Set the global varible |opt.query-response-time-stats| to ``ON``:

   .. code-block:: sql
		   
      mysql> SET GLOBAL query_response_time_stats=ON;

.. admonition:: Related Information: |percona-server| Documentation

      - |opt.query-response-time-stats|: https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#query_response_time_stats
      - Response time distribution: https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html#installing-the-plugins

.. include:: .res/replace.txt

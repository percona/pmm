.. _conf-mysql-requirements:

MySQL requirements
********************************************************************************

|pmm| supports all commonly used variants of |mysql|, including
|percona-server|, |mariadb|, and |amazon-rds|.  To prevent data loss and
performance issues, |pmm| does not automatically change |mysql| configuration.
However, there are certain recommended settings that help maximize monitoring
efficiency. These recommendations depend on the variant and version of |mysql|
you are using, and mostly apply to very high loads.

|pmm| can collect query data either from the |slow-query-log| or from
|performance-schema|.  The |slow-query-log| provides maximum details, but can
impact performance on heavily loaded systems. On |percona-server| the query
sampling feature may reduce the performance impact.

|performance-schema| is generally better for recent versions of other |mysql|
variants. For older |mysql| variants, which have neither sampling, nor
|performance-schema|, configure logging only slow queries.

.. note:: |mysql| with too many tables can lead to PMM Server overload due to
   because of streaming too much time series data. It can also lead to too many
   queries from ``mysqld_exporter`` and extra load on |mysql|. Therefore PMM
   Server disables most consuming ``mysqld_exporter`` collectors automatically
   if there are more than 1000 tables.

You can add configuration examples provided in this guide to :file:`my.cnf` and
restart the server or change variables dynamically using the following syntax:

.. code-block:: sql

   SET GLOBAL <var_name>=<var_value>

The following sample configurations can be used depending on the variant and
version of |mysql|:

* If you are running |percona-server| (or |xtradb-cluster|), configure the
  |slow-query-log| to capture all queries and enable sampling. This will provide
  the most amount of information with the lowest overhead.

  ::

   log_output=file
   slow_query_log=ON
   long_query_time=0
   log_slow_rate_limit=100
   log_slow_rate_type=query
   log_slow_verbosity=full
   log_slow_admin_statements=ON
   log_slow_slave_statements=ON
   slow_query_log_always_write_time=1
   slow_query_log_use_global_control=all
   innodb_monitor_enable=all
   userstat=1

* If you are running |mysql| 5.6+ or |mariadb| 10.0+, configure
  :ref:`perf-schema`.

  ::

   innodb_monitor_enable=all
   performance_schema=ON

* If you are running |mysql| 5.5 or |mariadb| 5.5, configure logging only slow
  queries to avoid high performance overhead.

  .. note:: This may affect the quality of monitoring data gathered by
            |abbr.qan|.

  ::

   log_output=file
   slow_query_log=ON
   long_query_time=0
   log_slow_admin_statements=ON
   log_slow_slave_statements=ON

.. _pmm.conf-mysql.user-account.creating:

`Creating a MySQL User Account to Be Used with PMM <services-mysql.html#pmm-conf-mysql-user-account-creating>`_
=========================================================================================================================

When adding a |mysql| instance to monitoring, you can specify the |mysql| server
superuser account credentials.  However, monitoring with the superuser account
is not secure. It's better to create a user with only the necessary privileges
for collecting data.

.. seealso::

   Using the |pmm-admin.add| command to add a monitoring service
      :ref:`pmm-admin.add-mysql-metrics`

For example can set up the ``pmm`` user manually with necessary privileges and
pass its credentials when adding the instance.

To enable complete |mysql| instance monitoring, a command similar to the
following is recommended:

.. prompt:: bash

   sudo pmm-admin add mysql --username root --password root

Of course this user should have necessary privileges for collecting data. If
the ``pmm`` user already exists, you can grant the required privileges as
follows:

.. code-block:: sql

   GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@' localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
   GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@'localhost';


For more information, run as root
|pmm-admin.add|
|opt.mysql|
|opt.help|.

.. include:: .res/replace.txt

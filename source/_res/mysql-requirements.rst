PMM supports all commonly used variants of MySQL, including
Percona Server, MariaDB, and Amazon RDS.  To prevent data loss and
performance issues, PMM does not automatically change MySQL configuration.
However, there are certain recommended settings that help maximize monitoring
efficiency. These recommendations depend on the variant and version of MySQL
you are using, and mostly apply to very high loads.

PMM can collect query data either from the *slow query log* or from
*Performance Schema*.  The *slow query log* provides maximum details, but can
impact performance on heavily loaded systems. On Percona Server the query
sampling feature may reduce the performance impact.

*Performance Schema* is generally better for recent versions of other MySQL
variants. For older MySQL variants, which have neither sampling, nor
*Performance Schema*, configure logging only slow queries.

.. note:: MySQL with too many tables can lead to PMM Server overload due to the
   streaming of too much time series data. It can also lead to too many queries
   from ``mysqld_exporter`` causing extra load on MySQL. Therefore PMM Server
   disables most consuming ``mysqld_exporter`` collectors automatically if
   there are more than 1000 tables.

You can add configuration examples provided below to :file:`my.cnf` and
restart the server or change variables dynamically using the following syntax:

.. code-block:: sql

   SET GLOBAL <var_name>=<var_value>

The following sample configurations can be used depending on the variant and
version of MySQL:

* If you are running Percona Server (or XtraDB Cluster), configure the
  *slow query log* to capture all queries and enable sampling. This will
  provide the most amount of information with the lowest overhead.

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

* If you are running MySQL 5.6+ or MariaDB 10.0+, configure
  :ref:`perf-schema`.

  ::

   innodb_monitor_enable=all
   performance_schema=ON

* If you are running MySQL 5.5 or MariaDB 5.5, configure logging only slow
  queries to avoid high performance overhead.

  .. note:: This may affect the quality of monitoring data gathered by
            QAN (Query Analytics).

  ::

   log_output=file
   slow_query_log=ON
   long_query_time=0
   log_slow_admin_statements=ON
   log_slow_slave_statements=ON

Creating a MySQL User Account to Be Used with PMM
=========================================================================================================================

When adding a MySQL instance to monitoring, you can specify the MySQL
server superuser account credentials.  However, monitoring with the superuser
account is not advised. It's better to create a user with only the necessary
privileges for collecting data.

As an example, the user ``pmm`` can be created manually with the necessary
privileges and pass its credentials when adding the instance.

To enable complete MySQL instance monitoring, a command similar to the
following is recommended:

.. prompt:: bash

   sudo pmm-admin add mysql --username pmm --password <password>

Of course this user should have necessary privileges for collecting data. If
the ``pmm`` user already exists, you can grant the required privileges as
follows:

.. code-block:: sql

   CREATE USER 'pmm'@'localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
   GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';
                
.. seealso::

      :ref:`pmm-admin.add-mysql-metrics` - Using the ``pmm-admin add`` command
      to add a monitoring service


For more information, run: ``pmm-admin add mysql --help``



.. _perf-schema:

##############################
Configuring Performance Schema
##############################

The default source of query data for PMM is the *slow query log*.  It is
available in MySQL 5.1 and later versions.  Starting from MySQL 5.6
(including Percona Server 5.6 and later), you can choose to parse query data
from the *Performance Schema* instead of *slow query log*.  Starting from MySQL
5.6.6, *Performance Schema* is enabled by default.

*Performance Schema* is not as data-rich as the *slow query log*, but it has all the
critical data and is generally faster to parse. If you are not running
Percona Server (which supports sampling for the slow query log), then *Performance Schema* is a better alternative.

.. note:: Use of the performance schema is off by default in MariaDB 10.x.

To use *Performance Schema*, set the ``performance_schema`` variable to ``ON``:

.. code-block:: sql

   SHOW VARIABLES LIKE 'performance_schema';

.. code-block:: text

   +--------------------+-------+
   | Variable_name      | Value |
   +--------------------+-------+
   | performance_schema | ON    |
   +--------------------+-------+

If this variable is not set to **ON**, add the the following lines to the
MySQL configuration file ``my.cnf`` and restart MySQL:

.. code-block:: text

   [mysql]
   performance_schema=ON

If you are running a custom Performance Schema configuration, make sure that the
``statements_digest`` consumer is enabled:

.. code-block:: sql

   select * from setup_consumers;

.. code-block:: text

   +----------------------------------+---------+
   | NAME                             | ENABLED |
   +----------------------------------+---------+
   | events_stages_current            | NO      |
   | events_stages_history            | NO      |
   | events_stages_history_long       | NO      |
   | events_statements_current        | YES     |
   | events_statements_history        | YES     |
   | events_statements_history_long   | NO      |
   | events_transactions_current      | NO      |
   | events_transactions_history      | NO      |
   | events_transactions_history_long | NO      |
   | events_waits_current             | NO      |
   | events_waits_history             | NO      |
   | events_waits_history_long        | NO      |
   | global_instrumentation           | YES     |
   | thread_instrumentation           | YES     |
   | statements_digest                | YES     |
   +----------------------------------+---------+
   15 rows in set (0.00 sec)

.. important::

   *Performance Schema* instrumentation is enabled by default in MySQL 5.6.6 and
   later versions. It is not available at all in MySQL versions prior to 5.6.

   If certain instruments are not enabled, you will not see the corresponding
   graphs in the :ref:`dashboard-mysql-performance-schema` dashboard.  To enable
   full instrumentation, set the option ``--performance_schema_instrument`` to
   ``'%=on'`` when starting the MySQL server.

   .. code-block:: bash

      mysqld --performance-schema-instrument='%=on'

   This option can cause additional overhead and should be used with care.

If the instance is already running, configure the QAN agent to collect data
from *Performance Schema*:

1. Open the *PMM Query Analytics* dashboard.

2. Click the *Settings* button.

3. Open the *Settings* section.

4. Select ``Performance Schema`` in the *Collect from* drop-down list.

5. Click *Apply* to save changes.

If you are adding a new monitoring instance with the ``pmm-admin`` tool, use the
``--query-source`` *perfschema* option:

Run this command as root or by using the ``sudo`` command

.. code-block:: bash

   pmm-admin add mysql --username=pmm --password=pmmpassword --query-source='perfschema' ps-mysql 127.0.0.1:3306

For more information, run ``pmm-admin add mysql --help``.


.. seealso::

   `MySQL Server 5.7 Documentation: --performance_schema_instrument <https://dev.mysql.com/doc/refman/5.7/en/performance-schema-options.html#option_mysqld_performance-schema-instrument>`__

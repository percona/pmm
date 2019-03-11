.. _perf-schema:

`Configuring Performance Schema <perf-schema>`_
===================================================

The default source of query data for |pmm| is the |slow-query-log|.  It is
available in |mysql| 5.1 and later versions.  Starting from |mysql| 5.6
(including |percona-server| 5.6 and later), you can choose to parse query data
from the |perf-schema| instead of |slow-query-log|.  Starting from |mysql|
5.6.6, |perf-schema| is enabled by default.

|perf-schema| is not as data-rich as the |slow-query-log|, but it has all the
critical data and is generally faster to parse. If you are not running
|percona-server| (which supports :ref:`sampling for the slow query log
<pmm.conf-mysql.slow-log-settings>`), then |performance-schema| is a better alternative.

To use |perf-schema|, set the ``performance_schema`` variable to ``ON``:

.. include:: .res/code/show-variables.like.performance-schema.txt

If this variable is not set to **ON**, add the the following lines to the
|mysql| configuration file |my.cnf| and restart |mysql|:

.. include:: .res/code/my-conf.mysql.performance-schema.txt

If you are running a custom Performance Schema configuration, make sure that the
``statements_digest`` consumer is enabled:

::

 mysql> select * from setup_consumers;
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

   |perf-schema| instrumentation is enabled by default in |mysql| 5.6.6 and
   later versions. It is not available at all in |mysql| versions prior to 5.6.

   If certain instruments are not enabled, you will not see the corresponding
   graphs in the :ref:`dashboard.mysql-performance-schema` dashboard.  To enable
   full instrumentation, set the option |opt.performance-schema-instrument| to
   ``'%=on'`` when starting the |mysql| server.

   .. code-block:: bash

      $ mysqld --performance-schema-instrument='%=on'

   This option can cause additional overhead and should be used with care.

   .. seealso::

      |mysql| Documentation: |opt.performance-schema-instrument| option
         https://dev.mysql.com/doc/refman/5.7/en/performance-schema-options.html#option_mysqld_performance-schema-instrument

If the instance is already running, configure the |qan| agent to collect data
from |perf-schema|:

1. Open the |qan.name| dashboard.
#. Click the |gui.settings| button.
#. Open the |gui.settings| section.
#. Select |opt.performance-schema| in the |gui.collect-from| drop-down list.
#. Click |gui.apply| to save changes.

If you are adding a new monitoring instance with the |pmm-admin| tool, use the
|opt.query-source| *perfschema* option:

|tip.run-this.root|

.. include:: .res/code/pmm-admin.add.mysql.user.password.create-user.query-source.txt
		   
For more information, run
|pmm-admin.add|
|opt.mysql|
|opt.help|.

.. include:: .res/replace.txt

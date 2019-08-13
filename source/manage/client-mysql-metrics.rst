.. _pmm-admin.add-mysql-metrics:

`Adding MySQL Service Monitoring <pmm-admin.add-mysql-metrics>`_
================================================================================

You then add MySQL services (Metrics and Query Analytics) with the following command:

.. _pmm-admin.add-mysql-metrics.usage:

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mysql --query-source='slowlog' --username=pmm --password=pmm

where username and password are credentials for the monitored MySQL access,
which will be used locally on the database host. Additionally, a service name
can be appended to the command line parameters, otherwise it will be generated 
automatically as ``<node>-mysql``.

The output of this command may look as follows:

.. code-block:: text

   # pmm-admin add mysql --query-source='slowlog' --username=pmm --password=pmm sl-mysql
   MySQL Service added.
   Service ID  : /service_id/a89191d4-7d75-44a9-b37f-a528e2c4550f
   Service name: ps-mysql

.. note:: There are two possible sources for query metrics provided by MySQL to
   get data for the Query Analytics: the `Slow Log <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/conf-mysql-slow-log.html#conf-mysql-slow-log>`_ and the `Performance Schema <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/conf-mysql-perf-schema.html#perf-schema>`_. The ``--query-source`` option can be
   used to specify it, either as ``slowlog`` (it is also used by default if nothing specified) or as ``perfschema``::

     pmm-admin add mysql --username=pmm --password=pmm --query-source='perfschema' 127.0.0.1:3306

After this you can view MySQL metrics or examine the added node on the new PMM Inventory Dashboard.

.. only:: showhidden

	.. note:: It should be able to detect the local |pmm-client| name,
	   but you can also specify it explicitly as an argument.

	.. _pmm-admin.add-mysql-metrics.options:

	.. rubric:: OPTIONS

	The following options can be used with the |opt.mysql-metrics| alias:

	|opt.create-user|
	  Create a dedicated |mysql| user for |pmm-client| (named ``pmm``).

	|opt.create-user-maxconn|
	  Specify maximum connections for the dedicated |mysql| user (default is 10).

	|opt.create-user-password|
	  Specify password for the dedicated |mysql| user.

	|opt.defaults-file|
	  Specify the path to :file:`my.cnf`.

	|opt.disable-binlogstats|
	  Disable collection of binary log statistics.

	|opt.disable-processlist|
	  Disable collection of process state metrics.

	|opt.disable-tablestats|
	  Disable collection of table statistics.

	|opt.disable-table-stats-limit|
	  Specify the maximum number of tables
	  for which collection of table statistics is enabled
	  (by default, the limit is 1 000 tables).

	|opt.disable-userstats|
	  Disable collection of user statistics.

	|opt.force|
	  Force to create or update the dedicated |mysql| user.

	|opt.host|
	  Specify the |mysql| host name.

	|opt.password|
	  Specify the password for |mysql| user with admin privileges.

	|opt.port|
	  Specify the |mysql| instance port.

	|opt.socket|
	  Specify the |mysql| instance socket file.

	|opt.user|
	  Specify the name of |mysql| user with admin privileges.

	You can also use
	:ref:`global options that apply to any other command <pmm-admin.options>`,
	as well as
	:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

	.. seealso::

	   Default ports
	      :ref:`Ports <Ports>` in :ref:`pmm.glossary-terminology-reference`

	   More information about |qan.name|
	      :ref:`pmm.qan`

	.. _pmm-admin.add-mysql-metrics.detailed-description:

	.. rubric:: DETAILED DESCRIPTION

	When adding the |mysql| metrics monitoring service, the |pmm-admin| tool
	attempts to automatically detect the local |mysql| instance and |mysql|
	superuser credentials.  You can use options to provide this information, if it
	cannot be detected automatically.

	You can also specify the |opt.create-user| option to create a dedicated ``pmm``
	user on the |mysql| host that you want to monitor.  This user will be given all
	the necessary privileges for monitoring, and is recommended over using the
	|mysql| superuser.

	For example, to set up remote monitoring of |mysql| metrics on a server located
	at 192.168.200.3, use a command similar to the following:

	.. _code.pmm-admin.add-mysql-metrics.user.password.host.create-user:

	.. include:: ../.res/code/pmm-admin.add.mysql-metrics.user.password.host.create-user.txt

	For more information, run
	|pmm-admin.add|
	|opt.mysql-metrics|
	|opt.help|.

	.. seealso::

	   How to set up |mysql| for monitoring?
	      :ref:`conf-mysql`


.. include:: ../.res/replace.txt

.. _pmm-admin.add-mysql-metrics:

`Adding MySQL Service Monitoring <pmm-admin.add-mysql-metrics>`_
================================================================================

You then add MySQL services (Metrics and Query Analytics) with the following command:

.. _pmm-admin.add-mysql-metrics.usage:

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm

where username and password are credentials for the monitored MySQL access,
which will be used locally on the database host. Additionally, two positional
arguments can be appended to the command line flags: a service name to be used
by PMM, and a service address. If not specified, they are substituted
automatically as ``<node>-mysql`` and ``127.0.0.1:3306``.

The command line and the output of this command may look as follows:

.. code-block:: text

   # pmm-admin add mysql --query-source=slowlog --username=pmm --password=pmm sl-mysql 127.0.0.1:3306
   MySQL Service added.
   Service ID  : /service_id/a89191d4-7d75-44a9-b37f-a528e2c4550f
   Service name: sl-mysql

.. note:: There are two possible sources for query metrics provided by MySQL to
   get data for the Query Analytics: the `Slow Log <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/conf-mysql-slow-log.html#conf-mysql-slow-log>`_ and the `Performance Schema <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/conf-mysql-perf-schema.html#perf-schema>`_. The ``--query-source`` option can be
   used to specify it, either as ``slowlog`` (it is also used by default if nothing specified) or as ``perfschema``::

     pmm-admin add mysql --username=pmm --password=pmm --query-source=perfschema ps-mysql 127.0.0.1:3306

Beside positional arguments shown above you can specify service name and
service address with the following flags: ``--service-name``, ``--host`` (the
hostname or IP address of the service), and ``--port`` (the port number of the
service). If both flag and positional argument are present, flag gains higher
priority. Here is the previous example modified to use these flags::

     pmm-admin add mysql --username=pmm --password=pmm --service-name=ps-mysql --host=127.0.0.1 --port=3306 

.. note:: It is also possible to add MySQL instance using UNIX socket with use
   of a special ``--socket`` flag followed with the path to a socket without
   username, password and network type::

      pmm-admin add mysql --socket=/var/path/to/mysql/socket

After adding the service you can view MySQL metrics or examine the added node
on the new PMM Inventory Dashboard.



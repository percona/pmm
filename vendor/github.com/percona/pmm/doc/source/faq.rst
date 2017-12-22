.. _faq:

================================================================================
Frequently Asked Questions
================================================================================

.. contents::
   :local:
   :depth: 1

How can I contact the developers?
================================================================================

The best place to discuss PMM with developers and other community members
is the `community forum <https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management>`_.

If you would like to report a bug,
use the `PMM project in JIRA <https://jira.percona.com/projects/PMM>`_.

.. _sys-req:

What are the minimum system requirements for PMM?
================================================================================

* |pmm-server|

  Any system which can run Docker version 1.12.6 or later.

  It needs roughly 1 GB of storage for each monitored database node
  with data retention set to one week.

  .. note:: By default, :ref:`retention <data-retention>`
     is set to 30 days for Metrics Monitor
     and to 8 days for Query Analytics.
     Also consider :ref:`disabling table statistics <performance-issues>`,
     which can greatly decrease Prometheus database size.

  Minimum memory is 2 GB for one monitored database node,
  but it is not linear when you add more nodes.
  For example, data from 20 nodes should be easily handled with 16 GB.

* |pmm-client|

  Any modern 64-bit Linux distribution.
  It is tested on the latest versions of
  Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux.

  Minimum 100 MB of storage is required for installing the |pmm-client| package.
  With good constant connection to |pmm-server|, additional storage is not
  required.  However, the client needs to store any collected data that it is
  not able to send over immediately, so additional storage may be required if
  connection is unstable or throughput is too low.

.. _metrics_memory:

How to control memory consumption for PMM?
================================================================================

By default, Prometheus in |pmm-server| uses up to 768 MB of memory for storing
the most recently used data chunks.  Depending on the amount of data coming into
Prometheus, you may require a higher limit to avoid throttling data ingestion,
or allow less memory consumption if it is needed for other processes.

You can control the allowed memory consumption for Prometheus
by passing the ``METRICS_MEMORY`` environment variable
when :ref:`creating and running the PMM Server container <server-container>`.
To set the environment variable, use the ``-e`` option.
The value must be passed in kilobytes.
For example, to set the limit to 4 GB of memory::

 -e METRICS_MEMORY=4194304

.. note:: The limit affects only memory reserved for data chunks.
   Actual RAM usage by Prometheus is higher.
   It is recommended to set this limit to roughly 2/3 of the total memory
   that you are planning to allow for Prometheus.
   So in the previous example, if you set the limit to 4 GB,
   then Prometheus will use up to 6 GB of memory.

.. _data-retention:

How to control data retention for PMM?
================================================================================

By default, Prometheus stores time-series data for 30 days,
and QAN stores query data for 8 days.

Depending on available disk space and your requirements,
you may need to adjust data retention time.

You can control data retention by passing the :term:`METRICS_RETENTION
<METRICS_RETENTION (Option)>` and :term:`QUERIES_RETENTION
<QUERIES_RETENTION (Option)>` environment variables when
:ref:`creating and running the PMM Server container
<server-container>`.  To set environment variables, use the ``-e``
option.  The value is passed as a combination of hours, minutes, and
seconds.  For example, the default value of 30 days for
``METRICS_RETENTION`` is ``720h0m0s``.  You probably do not need to be
more precise than the number hours, so you can discard the minutes and
seconds.  For example, to decrease the retention period for
|prometheus| to 8 days::

-e METRICS_RETENTION=192h

.. seealso::

   Metrics retention

      :term:`METRICS_RETENTION <METRICS_RETENTION (Option)>`

   Queries retention

      :term:`QUERIES_RETENTION <QUERIES_RETENTION (Option)>`

.. _service-location:

Where are the services created by PMM Client?
================================================================================

When you add a monitoring instance using the |pmm-admin| tool,
it creates a corresponding service.
The name of the service has the following syntax:
``pmm-<type>-<port>``

For example: ``pmm-mysql-metrics-42002``.

The location of the services depends on the service manager:

+-----------------+-----------------------------+
| Service manager | Service location            |
+=================+=============================+
| ``systemd``     | :file:`/etc/systemd/system/`|
+-----------------+-----------------------------+
| ``upstart``     | :file:`/etc/init/`          |
+-----------------+-----------------------------+
| ``systemv``     | :file:`/etc/init.d/`        |
+-----------------+-----------------------------+

To see which service manager is used on your system,
run as root |pmm-admin.info|.

Where is DSN stored?
================================================================================

Every service created by |pmm-admin| when you add a monitoring instance gets a
DSN from the credentials provided, auto-detected, or created (when adding the
instance with the |opt.create-user| option).

For MySQL and MongoDB metrics instances (|opt.mysql-metrics| and
|opt.mongodb-metrics| services), the DSN is stored with the corresponding
service files.  For more information, see :ref:`service-location`.

For QAN instances (|opt.mysql-queries| service), the DSN is stored in local
configuration files under :file:`/usr/local/percona/qan-agent`.

Also, a sanitized copy of DSN (without the passowrd) is stored in Consul API for
information purposes (used by the |pmm-admin.list| command).

Where are PMM Client log files located?
================================================================================

Every service created by |pmm-admin| when you add a monitoring instance has a
separate log file located in :file:`/var/log/`.  The file names have the
following syntax: ``pmm-<type>-<port>.log``.

For example, the log file for the MySQL QAN monitoring service is
:file:`/var/log/pmm-mysql-queries-0.log`.

You can view all available monitoring instance types and corresponding ports
using the |pmm-admin.list| command.  For more information, see
:ref:`pmm-admin.list`.

How often are nginx logs in PMM Server rotated?
================================================================================

|pmm-server| runs ``logrotate`` to rotate nginx logs on a daily basis
and keep up to 10 latest log files.

.. _performance-issues:

What are common performance considerations?
================================================================================

If a MySQL server has a lot of schemas or tables,
it is recommended to disable per table metrics when adding the instance:

.. prompt:: bash

   sudo pmm-admin add mysql --disable-tablestats

.. note:: Table statistics are disabled automatically
   if there are over 1 000 tables.

For more information, run as root
|pmm-admin.add|
|opt.mysql|
|opt.help|.

Can I stop all services at once?
================================================================================

Yes, you can use |pmm-admin| to start and stop either individual services
that correspond to the added monitoring instances,
or all of them at once.

To stop all services:

.. prompt:: bash

   sudo pmm-admin stop --all

To start all services:

.. prompt:: bash

   sudo pmm-admin start --all

For more information about starting and stopping services,
see :ref:`pmm-admin.start`.

You can view all available monitoring instances
and the states of the corresponding services
using the |pmm-admin.list| command.
For more information, see :ref:`pmm-admin.list`.

.. _privileges:

What privileges are required to monitor a |mysql| instance?
================================================================================

When adding |mysql| instance to monitoring,
you can specify the |mysql| server superuser account credentials,
which has all privileges.
However, monitoring with the superuser account is not secure.
If you also specify the |opt.create-user| option,
it will create a user with only the necessary privileges for collecting data.

You can also set up the ``pmm`` user manually with necessary privileges
and pass its credentials when adding the instance.

To enable complete |mysql| instance monitoring,
a command similar to the following is recommended:

.. prompt:: bash

   sudo pmm-admin add mysql --user root --password root --create-user

The superuser credentials are required only to set up the ``pmm`` user with
necessary privileges for collecting data.  If you want to create this user
yourself, the following privileges are required:

.. code-block:: sql

   GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@' localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
   GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@' localhost';

If the ``pmm`` user already exists,
simply pass its credential when you add the instance:

.. prompt:: bash

   sudo pmm-admin add mysql --user pmm --password pass

For more information, run as root
|pmm-admin.add|
|opt.mysql|
|opt.help|.

Can I monitor multiple |mysql| instances?
================================================================================

Yes, you can add multiple |mysql| instances to be monitored from one
|pmm-client|.  In this case, you will need to provide a distinct port and socket
for each instance using the |opt.port| and |opt.socket| parameters, and specify
a unique name for each instance (by default, it uses the name of the
|pmm-client| host).

For example, if you are adding complete MySQL monitoring
for two local |mysql| servers,
the commands could look similar to the following:

.. code-block:: bash

   $ sudo pmm-admin add mysql --user root --password root --create-user --port 3001 instance-01
   $ sudo pmm-admin add mysql --user root --password root --create-user --port 3002 instance-02

For more information, run
|pmm-admin.add|
|opt.mysql|
|opt.help|.

Can I rename instances?
================================================================================

You can remove any monitoring instance as described in :ref:`pmm-admin.rm`
and then add it back with a different name.

When you remove a monitoring service, previously collected data remains
available in |grafana|.  However, the metrics are tied to the instance name.  So
if you add the same instance back with a different name, it will be considered a
new instance with a new set of metrics.  So if you are re-adding an instance and
want to keep its previous data, add it with the same name.

.. _service-port:

Can I use non-default ports for instances?
================================================================================

When you add an instance with the |pmm-admin| tool,
it creates a corresponding service that listens on a predefined client port:

+--------------------+----------------------+-------+
| General OS metrics | ``linux:metrics``    | 42000 |
+--------------------+----------------------+-------+
| MySQL metrics      | ``mysql:metrics``    | 42002 |
+--------------------+----------------------+-------+
| MongoDB metrics    | ``mongodb:metrics``  | 42003 |
+--------------------+----------------------+-------+
| ProxySQL metrics   | ``proxysql:metrics`` | 42004 |
+--------------------+----------------------+-------+

If a default port for the service is not available, |pmm-admin| automatically
chooses a different port.

If you want to assign a different port, use the |opt.service-port| option
when :ref:`adding instances <pmm-admin.add>`.

.. _troubleshoot-connection:

How to troubleshoot communication issues between PMM Client and PMM Server?
================================================================================

When :ref:`using Docker <run-server-docker>`, the container is constrained by
the host-level routing and firewall rules.  For example, your hosting provider
might have default *iptables* rules on their hosts that block communication
between |pmm-server| and |pmm-client|, resulting in *DOWN* targets in
Prometheus.  If this happens, check firewall and routing settings on the Docker
host.

Troubleshooting Tips
--------------------------------------------------------------------------------

If you encounter communication issues, try the following:

* Check ``netstat`` on |pmm-client| to see what state the connections are in
* Run ``curl www.google.com`` to see if you get a reply
* Try pinging the route from inside the container

.. _metrics-resolution:

What resolution is used for metrics?
================================================================================

The |opt.mysql-metrics| service collects metrics with different resolutions (1
second, 5 seconds, and 60 seconds)

The |opt.linux-metrics| and |opt.mongodb-metrics| services are set up to collect
metrics with 1 second resolution.

In case of bad network connectivity between |pmm-server| and |pmm-client| or
between |pmm-client| and the database server it is monitoring, scraping every
second may not be possible when latency is higher than 1 second.  You can change
the minimum resolution for metrics by passing the ``METRICS_RESOLUTION``
environment variable when :ref:`creating and running the PMM Server container
<server-container>`.  To set this environment variable, use the ``-e`` option.
The values can be between *1s* (default) and *5s*.  If you set a higher value,
|prometheus| will not start.

For example, to set the minimum resolution to 3 seconds:

:command:`-e METRICS_RESOLUTION=3s`

.. note:: Consider increasing minimum resolution
   when |pmm-server| and |pmm-client| are on different networks,
   or when :ref:`amazon-rds`.

Why do I get ``Failed ReadTopologyInstance`` error when adding MySQL host to Orchestrator?
==========================================================================================

You need to create Orchestrator's topology user on |mysql|
according to :ref:`this section <pmm/using.orchestrator>`.


.. include:: .res/replace/name.txt
.. include:: .res/replace/option.txt
.. include:: .res/replace/program.txt


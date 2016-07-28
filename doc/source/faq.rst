.. _faq:

==========================
Frequently Asked Questions
==========================

.. contents::
   :local:
   :depth: 1

What are the minimum system requirements for PMM?
=================================================

:PMM Server: Any system which can run Docker version 1.10 or later,
 and kernel 3.x-4.x

:PMM Client: Any modern 64-bit Linux distribution.
 We recommend the latest versions of
 Debian, Ubuntu, CentOS, and RedHat Enterprise Linux.

How to control memory consumption for Prometheus?
=================================================

By default, Prometheus in PMM Server uses up to 256 MB of memory
for storing the most recently used data chunks.
Depending on the amount of data coming into Prometheus,
you may require a higher limit to avoid throttling data ingestion,
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
   It is recommended to have at least three times more memory
   than the expected memory taken up by data chunks.

.. _data-retention:

How to control data retention for Prometheus?
=============================================

By default, Prometheus in PMM Server stores time-series data for 30 days.
Depending on available disk space and your requirements,
you may need to adjust data retention time.

You can control data retention time for Prometheus
by passing the ``METRICS_RETENTION`` environment variable
when :ref:`creating and running the PMM Server container <server-container>`.
To set the environment variable, use the ``-e`` option.
The value is passed as a combination of hours, minutes, and seconds.
For example, the default value of 30 days is ``720h0m0s``.
You probably do not need to be more precise than the number hours,
so you can discard the minutes and seconds.
For example, to decrease the retention period to 8 days::

 -e METRICS_RETENTION=192h

.. _service-location:

Where are the services created by PMM Client?
=============================================

When you add a monitoring instance using the ``pmm-admin`` tool,
it creates a corresponding service.
The name of the service has the following syntax: ``pmm-<type>-exporter-<port>``

The location of the services depends on the service manager:

+-----------------+---------------------+-----------------------------+
| Service manager | Operating system    | Service location            |
+=================+=====================+=============================+
| ``systemd``     | CentOS 7 and RHEL 7 | :file:`/etc/systemd/system/`|
+-----------------+---------------------+-----------------------------+
| ``upstart``     | Debian and Ubuntu   | :file:`/etc/init/`          |
+-----------------+---------------------+-----------------------------+
| ``systemv``     | CentOS 6 and RHEL 6 | :file:`/etc/init.d/`        |
+-----------------+---------------------+-----------------------------+

To see which service manager is used on your system, run ``sudo pmm-admin info``.

Where is DSN stored?
====================

Every service created by ``pmm-admin`` when you add a monitoring instance,
gets a DSN from the credentials provided, auto-detected, or created
(when adding with the ``--create-user`` option).

For MySQL and MongoDB metrics instances (``mysql`` and ``mongodb`` types),
the DSN is stored with the corresponding service files.
For more information, see :ref:`service-location`.

For QAN instances (``queries`` type),
the DSN is stored in QAN API on *PMM Server*.

Also, a sanitized copy of DSN (without the passowrd)
is stored in Consul API for information purposes
(used by the ``pmm-admin list`` command).

Where are PMM Client log files located?
=======================================

Every service created by ``pmm-admin`` when you add a monitoring instance
has a separate log file located in :file:`/var/log/`.
The file names have the following syntax: ``pmm-<type>-exporter-<port>.log``

For example, the log file for the QAN monitoring service is
:file:`/var/log/pmm-queries-exporter-42001.log`.

You can view all available monitoring instance types and corresponding ports
using the ``pmm-admin list`` command.
For more information, see :ref:`pmm-admin-list`.

.. _performance-issues:

What are common performance considerations?
===========================================

If a MySQL server has a lot of schemas or tables,
it is recommended to disable per table metrics when adding the instance:

.. prompt:: bash

   sudo pmm-admin add mysql --disable-per-table-stats

If ``SELECT`` queries from ``information_schema`` tables slow down performance,
you can disable all metrics from it when adding the instance:

.. prompt:: bash

   sudo pmm-admin add mysql --disable-infoschema

For more information, run ``sudo pmm-admin add mysql --help``.

Can I stop all services at once?
================================

Yes, you can use ``pmm-admin`` to start and stop either individual services
that correspond to the added monitoring instances,
or all of them at once.

To stop all services:

.. prompt:: bash

   sudo pmm-admin stop --all

To start all services:

.. prompt:: bash

   sudo pmm-admin start --all

For more information about starting and stopping services,
see :ref:`pmm-admin-start`.

You can view all available monitoring instances
and the states of the corresponding services
using the ``pmm-admin list`` command.
For more information, see :ref:`pmm-admin-list`.

What privileges are required to monitor a MySQL instance?
=========================================================

When adding a :ref:`Query Analytics instance <pmm-admin-add-queries>`
or a :ref:`MySQL metrics instance <pmm-admin-add-mysql>`,
you can specify the MySQL server superuser account credentials,
which has all privileges.
However, monitoring with the superuser account is not secure.
If you also specify the ``--create-user`` option,
it will create a user with only the necessary privileges for collecting data.

You can also set up the user manually with necessary privileges
and pass its credentials when adding the instance.

User for QAN monitoring
-----------------------

To add a local QAN instance,
a command similar to the following is recommended:

.. prompt:: bash

   sudo pmm-admin add queries --user root --password root --create-user

The superuser credentials are required only to set up the ``pmm-queries`` user
with necessary privileges for collecting data.
If you want to create this user yourself, the following privileges are required::

 GRANT SELECT, PROCESS, SUPER ON *.* TO 'pmm-queries'@' localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 5;
 GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm-queries'@' localhost';

.. note:: If the query source for QAN is Performance Schema,
   the ``SUPER`` privilege is not required.
   By default, the slow query log is the preferred default.
   You can set the source with the ``--query-source perfschema`` option.
   In this case, if you also add the ``--create-user`` option,
   the ``SUPER`` privilege will not be granted to the ``pmm-queries`` user.

If the ``pmm-queries`` user already exists,
simply pass its credentials when you add the instance:

.. prompt:: bash

   sudo pmm-admin add queries --user pmm-queries --password pass

For more information, run ``sudo pmm-admin add queries --help``.

User for MySQL metrics monitoring
---------------------------------

To add a local MySQL metrics instance,
a command similar to the following is recommended:

.. prompt:: bash

   sudo pmm-admin add mysql --user root --password root --create-user

The superuser credential are required only to set up the ``pmm-mysql`` user
with necessary privileges for collecting data.
If you want to create this user yourself, the following privileges are required::

 GRANT PROCESS, REPLICATION CLIENT ON *.* TO 'pmm-mysql'@'localhost' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 5;
 GRANT SELECT ON performance_schema.* TO 'pmm-mysql'@'localhost';

If the ``pmm-mysql`` user already exists,
simply padd its credential when you add the instance:

.. prompt:: bash

   sudo pmm-admin add mysql --user pmm-mysql --password pass

For more information, run ``sudo pmm-admin add mysql --help``.

Can I monitor multiple MySQL instances?
=======================================

Yes, you can add multiple QAN and MySQL metrics monitoring instances
on a single *PMM Client* (that is, ``queries`` and ``mysql`` types).
In this case,
you will need to provide a distinct port and socket for each instance
using the ``--port`` and ``--socket`` variables,
and specify a unique name for each instance
(by default, it uses the host name).

For example, if you are adding QAN and MySQL metrics monitoring instances
for two local MySQL servers,
the commands could look similar to the following:

.. code-block:: bash

   $ sudo pmm-admin add queries --user root --password root --create-user --port 3001 instance-01
   $ sudo pmm-admin add queries --user root --password root --create-user --port 3002 instance-02
   $ sudo pmm-admin add mysql --user root --password root --create-user --port 3001 instance-01
   $ sudo pmm-admin add mysql --user root --password root --create-user --port 3002 instance-02

For more information, run ``sudo pmm-admin add queries --help``
or ``sudo pmm-admin add mysql --help``.

Can I rename instances?
=======================

You can remove any monitoring instance as described in :ref:`pmm-admin-rm`
and then add it back with a different name.

When you remove a general OS, MySQL, or MongoDB metrics monitoring instance,
previously collected data remains available in Grafana.
However, the metrics are tied to the instance name.
So if you add the same instance back with a different name,
it will be considered a new instance with a new set of metrics.
(this is true for ``os``, ``mysql``, and ``mongodb`` types).

When you remove a QAN instance (``queries`` type),
previously collected data will no longer be available after you add it back,
regardless of the name you use.

.. _service-port:

Can I use non-default ports for instances?
==========================================

When you add an instance with the ``pmm-admin`` tool,
it creates a corresponding service that listens on a predefined client port:

+--------------------+-------------+---------------------+
| General OS metrics | ``os``      | 42000               |
+--------------------+-------------+---------------------+
| Query analytics    | ``queries`` | 42001               |
+--------------------+-------------+---------------------+
| MySQL metrics      | ``mysql``   | 42002, 42003, 42004 |
+--------------------+-------------+---------------------+
| MongoDB metrics    | ``mongodb`` | 42005               |
+--------------------+-------------+---------------------+

If a default port for the service is not available,
``pmm-admin`` automatically chooses a different one.

If you want to assign a different port, use the ``--service-port`` option
when :ref:`adding instances <pmm-admin-add>`.

.. _metrics-resolution:

What resolution is used for metrics?
====================================

MySQL metrics instance uses three services,
which collect metrics with different resolutions
(1 second, 5 seconds, and 60 seconds)

MongoDB and OS instances are set up to collect metrics with 1 second resolution.

In case of bad network connectivity between *PMM Server* and *PMM Client*
or between *PMM Client* and the database server it is monitoring,
scraping every second may not be possible when latency is higher than 1 second.
You can change the minimum resolution for metrics
by passing the ``METRICS_RESOLUTION`` environment variable
when :ref:`creating and running the PMM Server container <server-container>`.
To set this environment variable, use the ``-e`` option.
The values can be between ``1s`` (default) and ``5s``.
If you set a higher value, Prometheus will not start.

For example, to set the minimum resolution to 3 seconds::

 -e METRICS_RESOLUTION=3s

.. note:: Consider increasing minimum resolution
   when *PMM Server* and *PMM Client* are on different networks,
   or when :ref:`amazon-rds`.


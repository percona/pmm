:orphan: true

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

.. rubric:: |pmm-server|

Any system which can run Docker version 1.12.6 or later.

It needs roughly 1 GB of storage for each monitored database node
with data retention set to one week.

.. note::

   By default, :ref:`retention <data-retention>` is set to 30 days for
   Metrics Monitor and to 8 days for Query Analytics.  Also consider
   :ref:`disabling table statistics <performance-issues>`, which can
   greatly decrease Prometheus database size.

Minimum memory is 2 GB for one monitored database node, but it is not
linear when you add more nodes.  For example, data from 20 nodes
should be easily handled with 16 GB.

.. rubric:: |pmm-client|

Any modern 64-bit Linux distribution. It is tested on the latest
versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux.

Minimum 100 MB of storage is required for installing the |pmm-client|
package.  With good constant connection to |pmm-server|, additional
storage is not required.  However, the client needs to store any
collected data that it is not able to send over immediately, so
additional storage may be required if connection is unstable or
throughput is too low.

.. _data-retention:

How to control data retention for PMM?
================================================================================

By default, |prometheus| stores time-series data for 30 days,
and QAN stores query data for 8 days.

Depending on available disk space and your requirements,
you may need to adjust data retention time.

You can control data retention by passing the :option:`METRICS_RETENTION` and :option:`QUERIES_RETENTION` environment variables when
:ref:`creating and running the PMM Server container
<server-container>`.  To set environment variables, use the ``-e``
option.  The value should be the number of hours, minutes, or
seconds. For example, the default value of 30 days for
|opt.metrics-retention| is ``720h``.  You probably do not need to be
more precise than the number hours, so you can discard the minutes and
seconds.  For example, to decrease the retention period for
|prometheus| to 8 days::

-e METRICS_RETENTION=192h

.. seealso::

   Metrics retention
      :option:`METRICS_RETENTION`

   Queries retention
      :option:`QUERIES_RETENTION`

.. _service-location:

Where are the services created by PMM Client?
================================================================================

When you add a monitoring instance using the |pmm-admin| tool, it creates a
corresponding service.  The name of the service has the following syntax:

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

See :ref:`pmm.conf-mysql.user-account.creating`.

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

There is a ``pmm-admin check-network`` command, which `checks connectivity <https://www.percona.com/doc/percona-monitoring-and-management/pmm-admin.html#pmm-admin-check-network>`_ between |pmm-client|
and |pmm-server| and presents the summary of this check in a human readable form.

Broken network connectivity may be caused by rather wide set of reasons.
Particularly, when :ref:`using Docker <run-server-docker>`, the container is
constrained by the host-level routing and firewall rules. For example, your
hosting provider might have default *iptables* rules on their hosts that block
communication between |pmm-server| and |pmm-client|, resulting in *DOWN* targets
in Prometheus. If this happens, check firewall and routing settings on the
Docker host.

Also |pmm| is able to generate a set of diagnostics data which can be examined
and/or shared with Percona Support to solve an issue faster. See details on how
to get collected logs `from PMM Server <https://www.percona.com/doc/percona-monitoring-and-management/deploy/index.html#deploy-pmm-diagnostics-for-support>`_ and `from PMM Client <https://www.percona.com/doc/percona-monitoring-and-management/pmm-admin.html#pmm-admin-diagnostics-for-support>`_.

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
   or when :ref:`pmm.amazon-rds`.

Why do I get ``Failed ReadTopologyInstance`` error when adding MySQL host to Orchestrator?
==========================================================================================

You need to create Orchestrator's topology user on |mysql|
according to :ref:`this section <pmm.using.orchestrator>`.

.. _pmm.deploying.server.virtual-appliance.root-password.setting:

How to set the root password when |pmm-server| is installed as a virtual appliance
====================================================================================================

With your virtual appliance set up, you need to set the root password for your
|pmm-server|. By default, the virtual machine is configured to enforce changing
the default password upon the first login.

.. figure:: .res/graphics/png/command-line.login.1.png

   Set the root password when logging in.

Run your virtual machine and when requested to log in, use the following
credentials:

:User: root
:Password: percona

The system immediately requests that you change your password. Note that, for
the sake of security, your password must not be trivial and pass at least the
dictionary check. If you do not provide your password within sixty seconds you
are automatically logged out. In this case use the default credentials to log in
again.

.. figure:: .res/graphics/png/command-line.login.3.png

   Set a new password and have full access to your system

After the new password is set you control this system as a superuser and
can make whaterver changes required.

.. important::

   You cannot access the root account if you access |pmm-server| using
   SSH or via the Web interface.

.. _pmm.pmm-server.experimental-version.installing:

How to install the experimental version of |pmm-server|?
================================================================================

If you would like to experiment with the latest development version using
|docker|, you may use the |opt.dev-latest| image. This version, however, is not
intended to be used in a production environment.

.. include:: .res/code/docker.pull.perconalab-pmm-server-dev-latest.txt

If you would like to experiment with the latest development version of
|pmm-server| |virtualbox| image, download the development version as follows:

.. include:: .res/code/wget.pmm-server-dev-latest-ova.txt

.. important:: 

   This is a development version which is not designed for a production
   environment.

.. seealso::

   Setting up |pmm-server| via |docker|
      :ref:`setup procedure <pmm.server.docker.setting-up>`
   Setting up |pmm-server| via |virtualbox|
      :ref:`pmm.deploying.server.ova.virtualbox.cli`

.. include:: .res/replace.txt


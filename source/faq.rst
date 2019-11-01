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
   Metrics Monitor and for Query Analytics.  Also consider
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

By default, both |prometheus| and QAN store time-series data for 30 days.

Depending on available disk space and your requirements,
you may need to adjust data retention time.

You can control data retention by passing the :option:`DATA_RETENTION`
environment variable when :ref:`creating and running the PMM Server container
<server-container>`.  To set environment variable, use the ``-e``
option.  The value should be the number of hours, and requires h suffix. 
For example, the default value of 30 days for |opt.metrics-retention| is
``720h``, but you can decrease the retention period for |prometheus| to 8 days
as follows::

-e DATA_RETENTION=192h

.. seealso::

   Metrics and queries retention
      :option:`DATA_RETENTION`

How often are nginx logs in PMM Server rotated?
================================================================================

|pmm-server| runs ``logrotate`` to rotate nginx logs on a daily basis
and keep up to 10 latest log files.

.. only:: showhidden

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

.. _privileges:

What privileges are required to monitor a |mysql| instance?
================================================================================

See :ref:`pmm.conf-mysql.user-account.creating`.

Can I monitor multiple service instances?
================================================================================

Yes, you can add multiple instances of |mysql| or some other service to be
monitored from one |pmm-client|. In this case, you will need to provide a
distinct port and socket for each instance, and specify a unique name for each
instance (by default, it uses the name of the |pmm-client| host).

For example, if you are adding complete MySQL monitoring for two local |mysql|
servers, the commands could look similar to the following:

.. code-block:: bash

   $ sudo pmm-admin add mysql --username root --password root instance-01 127.0.0.1:3001
   $ sudo pmm-admin add mysql --username root --password root instance-02 127.0.0.1:3002

For more information, run

.. code-block:: bash

   $ pmm-admin add mysql --help

Can I rename instances?
================================================================================

You can remove any monitoring instance as described in :ref:`pmm-admin.rm`
and then add it back with a different name.

When you remove a monitoring service, previously collected data remains
available in |grafana|.  However, the metrics are tied to the instance name.  So
if you add the same instance back with a different name, it will be considered a
new instance with a new set of metrics.  So if you are re-adding an instance and
want to keep its previous data, add it with the same name.

.. _troubleshoot-connection:

How to troubleshoot communication issues between PMM Client and PMM Server?
================================================================================

Broken network connectivity may be caused by rather wide set of reasons.
Particularly, when :ref:`using Docker <run-server-docker>`, the container is
constrained by the host-level routing and firewall rules. For example, your
hosting provider might have default *iptables* rules on their hosts that block
communication between |pmm-server| and |pmm-client|, resulting in *DOWN* targets
in Prometheus. If this happens, check firewall and routing settings on the
Docker host.

Also |pmm| is able to generate a set of diagnostics data which can be examined
and/or shared with Percona Support to solve an issue faster. You can get
collected logs from PMM Client using the ``pmm-admin summary`` command. 
Obtaining logs from PMM Server can be done `by specifying the
``https://<address-of-your-pmm-server>/logs.zip`` URL, or by clicking
the ``server logs`` link on the `Prometheus dashboard <https://www.percona.com/doc/percona-monitoring-and-management/2.x/dashboards/dashboard-prometheus.html>`_:

.. image:: .res/graphics/png/get-logs-from-prometheus-dashboard.png

.. only:: showhidden

	.. _metrics-resolution:

	What resolution is used for metrics?
	================================================================================

	The |opt.mysql-metrics| service collects metrics with different resolutions (5
	seconds, 5 seconds, and 60 seconds by default),

	The |opt.linux-metrics| and |opt.mongodb-metrics| services are set up to collect
	metrics with 1 second resolution.

	In case of bad network connectivity between |pmm-server| and |pmm-client| or
	between |pmm-client| and the database server it is monitoring, scraping every
	second may not be possible when latency is higher than 1 second.  You can change
	the minimum resolution for metrics by passing the ``METRICS_RESOLUTION``
	environment variable when :ref:`creating and running the PMM Server container
	<server-container>`. To set this environment variable, use the ``-e`` option.
	The values can be between *1s* and *5s* (default).  If you set a higher value,
	|prometheus| will not start.

	For example, to set the minimum resolution to 3 seconds:

	:command:`-e METRICS_RESOLUTION=3s`

	.. note:: Consider increasing minimum resolution
	   when |pmm-server| and |pmm-client| are on different networks,
	   or when :ref:`pmm.amazon-rds`.

.. only:: showhidden

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

.. include:: .res/replace.txt


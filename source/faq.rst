.. _faq:

##########################
Frequently Asked Questions
##########################

.. contents::
   :local:
   :depth: 1


*********************************
How can I contact the developers?
*********************************

The best place to discuss PMM with developers and other community members is the `community forum <https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management>`_.

To report a bug, visit the `PMM project in JIRA <https://jira.percona.com/projects/PMM>`_.



.. _sys-req:

*************************************************
What are the minimum system requirements for PMM?
*************************************************

.. rubric:: PMM Server

Any system which can run Docker version 1.12.6 or later.

It needs roughly 1 GB of storage for each monitored database node with data retention set to one week.

.. note::

   By default, :ref:`retention <data-retention>` is set to 30 days for Metrics Monitor and for Query Analytics.  You can consider :ref:`disabling table statistics <performance-issues>` to decrease the Prometheus database size.

The minimum memory requirement is 2 GB for one monitored database node.

.. note::

   The increase in memory usage is not proportional to the number of nodes.  For example, data from 20 nodes should be easily handled with 16 GB.

.. rubric:: PMM Client

Any modern 64-bit Linux distribution. It is tested on the latest versions of Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux.

A minimum of 100 MB of storage is required for installing the PMM Client package.  With a good connection to PMM Server, additional storage is not required.  However, the client needs to store any collected data that it cannot dispatch immediately, so additional storage may be required if the connection is unstable or the throughput is low.
(Caching only applies to Query Analytics data; Prometheus data is never cached on the client side.)


.. _data-retention:
.. _how-to-control-data-retention-for-pmm:

**************************************
How to control data retention for PMM?
**************************************

By default, PMM stores time-series data for 30 days.
Depending on your available disk space and requirements, you may need to adjust the data retention time:

#. Select the PMM Settings dashboard in the main menu.

   .. image:: /_images/pmm-add-instance.png

#. In the *Settings* section, enter the new data retention value in days.

   .. image:: /_images/pmm.settings_settings.png

#. Click *Apply changes*.



***********************************************
How often are NGINX logs in PMM Server rotated?
***********************************************

PMM Server runs ``logrotate`` on a daily basis to rotate NGINX logs and keeps up to ten of the most recent log files.



.. _privileges:

*********************************************************
What privileges are required to monitor a MySQL instance?
*********************************************************

.. code-block:: sql

   GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm'@'localhost';


*****************************************
Can I monitor multiple service instances?
*****************************************

You can add multiple instances of MySQL or some other service to be monitored from one PMM Client. In this case, you must provide a unique port and IP address, or a socket for each instance, and specify a unique name for each.  (If a name is not provided, PMM uses the name of the PMM Client host.)

For example, to add complete MySQL monitoring for two local MySQL servers, the commands would be:

.. code-block:: bash

   $ sudo pmm-admin add mysql --username root --password root instance-01 127.0.0.1:3001
   $ sudo pmm-admin add mysql --username root --password root instance-02 127.0.0.1:3002

For more information, run:

.. code-block:: bash

   $ pmm-admin add mysql --help



***********************
Can I rename instances?
***********************

You can remove any monitoring instance and then add it back with a different name.  ( see :ref:`pmm-admin.rm`).

When you remove a monitoring service, previously collected data remains available in Grafana.  However, the metrics are tied to the instance name.  So if you add the same instance back with a different name, it will be considered a new instance with a new set of metrics.  So if you are re-adding an instance and want to keep its previous data, add it with the same name.

*************************************************************************************
Can I add an AWS RDS MySQL or Aurora MySQL instance from a non-default AWS partition?
*************************************************************************************

By default, the RDS discovery works with the default ``aws`` partition. But you can switch to special regions, like the `GovCloud <https://aws.amazon.com/govcloud-us/>`_ one, with the alternative `AWS partitions <https://docs.aws.amazon.com/sdk-for-go/api/aws/endpoints/#pkg-constants>`_ (e.g. ``aws-us-gov``) adding them to the *Settings* via the PMM Server API (see :ref:`pmm-server-api`).

.. image:: /_images/aws-partitions-in-api.png

To specify other than the default value, or to use several, use the JSON Array syntax: ``["aws", "aws-cn"]``.



.. _troubleshoot-connection:

*****************************************************************************
How do I troubleshoot communication issues between PMM Client and PMM Server?
*****************************************************************************

Broken network connectivity may be due to many reasons.  Particularly, when :ref:`using Docker <run-server-docker>`, the container is constrained by the host-level routing and firewall rules. For example, your hosting provider might have default *iptables* rules on their hosts that block communication between PMM Server and PMM Client, resulting in *DOWN* targets in Prometheus. If this happens, check the firewall and routing settings on the Docker host.

PMM is also able to generate diagnostics data which can be examined and/or shared with Percona Support to help quickly solve an issue. You can get collected logs from PMM Client using the ``pmm-admin summary`` command.

Logs obtained in this way includes PMM Client logs and logs which were received from the PMM Server, stored separately in the ``client`` and ``server`` folders. The ``server`` folder also contains its own ``client`` subfolder with the self-monitoring client information collected on the PMM Server.

.. note:: Beginning with PMM version 2.4.0, there is an additional flag that enables the fetching of `pprof <https://github.com/google/pprof>`_ debug profiles and adds them to the diagnostics data. To enable, run ``pmm-admin summary --pprof``.

Obtaining logs from PMM Server can be done by specifying the ``https://<address-of-your-pmm-server>/logs.zip`` URL, or by clicking the ``server logs`` link on the :ref:`dashboard-prometheus`:

.. image:: /_images/get-logs-from-prometheus-dashboard.png



.. _metrics-resolution:

************************************
What resolution is used for metrics?
************************************

The default values are:

- Low: 60 seconds
- Medium: 10 seconds
- High: 5 seconds

(See :ref:`server-admin-gui-metrics-resolution`.)



.. _how-to-setup-alerting-with-grafana:

********************************
How do I set up Alerting in PMM?
********************************

When a monitored service metric reaches a defined threshold, PMM Server can trigger alerts for it either using the Grafana Alerting feature or by using an external Alertmanager, a high-performance solution developed by the Prometheus project to handle alerts sent by Prometheus.

With these methods you must configure alerting rules that define conditions under which an alert should be triggered, and the channel used to send the alert (e.g. email).

Alerting in Grafana allows attaching rules to your dashboard panels.  Grafana Alerts are already integrated into PMM Server and may be simpler to get set up.

Alertmanager allows the creation of more sophisticated alerting rules and can be easier to manage installations with a large number of hosts. This additional flexibility comes at the expense of simplicity.

.. note::

   We can only offer support for creating custom rules to Percona customers, so you should already have a working Alertmanager instance prior to using this feature.


.. seealso::

   - `Grafana Alerts overview <https://grafana.com/docs/grafana/latest/alerting/>`_

   - `Alertmanager <https://prometheus.io/docs/alerting/latest/alertmanager/#alertmanager>`_

   - `PMM Alerting with Grafana: Working with Templated Dashboards <https://www.percona.com/blog/2017/02/02/pmm-alerting-with-grafana-working-with-templated-dashboards/>`_

**********************************************************************
How do I use a custom Prometheus configuration file inside PMM Server?
**********************************************************************

Normally, PMM Server fully manages the `Prometheus configuration file <https://prometheus.io/docs/prometheus/latest/configuration/configuration/>`_.

However, some users may want to change the generated configuration to add additional scrape jobs, configure remote storage, etc.

From version 2.4.0, when ``pmm-managed`` starts the Prometheus file generation process, it tries to load the ``/srv/prometheus/prometheus.base.yml`` file first, to use it as a base for the ``prometheus.yml`` file.

.. note:: The ``prometheus.yml`` file can be regenerated by restarting the PMM Server container, or by using the ``SetSettings`` API call with an empty body (see :ref:`pmm-server-api`).

.. seealso:: `Extending PMM’s Prometheus Configuration <https://www.percona.com/blog/2020/03/23/extending-pmm-prometheus-configuration/>`_

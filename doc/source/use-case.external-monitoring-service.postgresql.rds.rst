:orphan: true

.. _use-case.external-monitoring-service.postgresql.rds:

Use case: Monitoring a |postgresql| database running on an |amazon-rds| instance
********************************************************************************

As of version 1.14.0 |pmm| supports |postgresql| `out-of-the-box <https://www.percona.com/doc/percona-monitoring-and-management/conf-postgres.html>`_. 

This example demonstrates how to start monitoring a |postgresql| host which is
installed on an |amazon-rds| instance.

.. important::

   This use case is limited to demonstrating the essential part of using
   external monitoring services of |pmm| and should be treated as an example. As
   such, it does not demostrate how to use the security features of |amazon-rds|
   or of the |prometheus| exporter being used.

.. contents::
   :local:
   
Set Up the |postgresql| Exporter
================================================================================

First, you need to enable an exporter for |postgresql| on the
computer where you have installed the |pmm-client| package with the
``by pmm-admin add postgresql`` command. More information on enabling and
configuring |postgresql| exporter can be found in the `detailed instruction <https://www.percona.com/doc/percona-monitoring-and-management/conf-postgres.html>`_.


Check Settings of Your |amazon-rds| Instance
================================================================================

Your |amazon-rds| instance where you have installed |postgresql| must be allowed
to communicate outside of the VPC hosting the DB instance. Select *Yes* in the
|gui.public-accessibility| field.

.. figure:: .res/graphics/png/amazon-rds.modify-db-instance.1.png

   Modify your |amazon-rds| instance and make it publicly accessible

.. seealso::

   More information about |amazon-rds|
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Welcome.html
   Using |postgresql| with |amazon-rds|
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_PostgreSQL.html
   Connecting to an |amazon-rds| DB instance running |postgresql|
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_ConnectToPostgreSQLInstance.html

Add monitoring service for |postgresql|
================================================================================

To make the metrics from your |amazon-rds| instance available to |pmm|, you need
to run |pmm-admin.add| command as follows:

|tip.run-this.root|

.. code-block:: bash 
    pmm-admin add postgresql --create-user --host=172.17.0.2 --password=ABC123 --port=5432 --user=postgres_exporter postgresql_rds01

The last parameter gives a distinct name to your host. If you do not specify a
custom instance name, the name of the host where you run |pmm-admin.add| is used
automatically.

.. code-block:: bash

   $ pmm-admin list
   pmm-admin 1.8.0

   PMM Server      | 127.0.0.1 
   Client Name     | percona
   Client Address  | 172.17.0.1 
   Service Manager | linux-systemd

   ...
   
   Job name  Scrape interval  Scrape timeout  Metrics path  Scheme  Target           Labels                       Health
   postgres  1m0s             10s             /metrics      http    172.17.0.1:9187  instance="postgresql_rds01"  DOWN

Viewing |postgresql| Metrics in |pmm|
================================================================================

Now, open |metrics-monitor| in your browser and select the
|dbd.advanced-data-exploration| dashboard either using the |gui.dashboard-dropdown|
or the |gui.insight| group of the navigation menu. Use the |gui.metric| field to
select the name of a metric. Note that postgresql specific metrics start with
*pg_*.

.. figure:: .res/graphics/png/metrics-monitor.advanced-data-exploration.1.png

   Using the |dbd.advanced-data-exploration| dashboard to select a |postgresql| metric.

.. seealso::

   How to add an external monitoring services to |pmm|
      :ref:`pmm.pmm-admin.external-monitoring-service.adding`

.. include:: .res/replace.txt

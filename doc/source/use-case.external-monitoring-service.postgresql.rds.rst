:orphan: true

.. _use-case.external-monitoring-service.postgresql.rds:

================================================================================
Use case: Monitoring a |postgresql| database running on an |amazon-rds| instance
================================================================================

|pmm| currently does not support |postgresql| out-of-the-box. However, you can
monitor your |postgresql| host by using external monitoring services.  The
external monitoring services only require that the appropriate |prometheus|
exporter be properly installed on the system where |pmm-admin| is available (see
section :ref:`install-client`).

This example demonstrates how to start monitoring a |postgresql| host which is
installed on an |amazon-rds| instance.

.. important::

   This use case is limited to demonstrating the essential part of using
   external monitoring services of |pmm| and should be treated as an example. As
   such, it does not demostrate how to use the security features of |amazon-rds|
   or of the |prometheus| exporter being used.
   
Set Up the |postgresql| Exporter
================================================================================

First, you need to install a |prometheus| exporter for |postgresql| on the
computer where you have installed the |pmm-client| package. This example uses
the |postgresql| exporter listed on the |prometheus| site:
https://github.com/wrouesnel/postgres_exporter. Note that this exporter requires
that the `Go <https://golang.org/>`_ programming language environment be
properly set up and configured. Alternatively, you may run the exporter from the
|docker| image as explained on the site.

.. seealso::

   |prometheus| Exporters and integrations
      https://prometheus.io/docs/instrumenting/exporters/
   Installing the Go programming language
      https://golang.org/doc/install

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
   
Read Metrics from the |postgresql| DB Instance
================================================================================

As suggested in the documentation of the |postgresql| exporter, we set the
:code:`DATA_SOURCE_NAME` variable and start the exporter.

Note that the following example disables **sslmode** which will make your system
less secure. It also uses |sudo| to demonstrate that the code should be run as
the *postgres* user. Before running this command make sure to |cd| into the
directory that contains the built :program:`postgresql_exporter` binary.

.. include:: .res/code/sh.org
   :start-after: +sudo.data-source-name.postgresql-exporter+
   :end-before: #+end-block

The |postgresql| exporter makes its metrics available on port 9187.

Add an external monitoring service for |postgresql|
================================================================================

To make the metrics from your |amazon-rds| instance available to |pmm|, you need
to run |pmm-admin.add| command as follows:

|tip.run-this.root|

.. code-block:: bash
   
   $ pmm-admin add external:service postgres --service-port=9187 postgres_rds01

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
|advanced-data-exploration| dashboard either using the |gui.dashboard-dropdown|
or the |gui.insight| group of the navigation menu. Use the |gui.metric| field to
select the name of a metric. Note that postgresql specific metrics start with
*pg_*.

.. figure:: .res/graphics/png/metrics-monitor.advanced-data-exploration.1.png

   Using the |advanced-data-exploration| dashboard to select a |postgresql| metric.

.. seealso::

   Adding external monitoring services to |PMM|
      :ref:`pmm/pmm-admin/external-monitoring-service.adding`

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/fragment.txt

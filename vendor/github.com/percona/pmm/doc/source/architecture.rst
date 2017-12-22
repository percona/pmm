.. _architecture:

================================================================================
|pmm.name| Architecture
================================================================================

The |pmm| platform is based on a client-server model that enables scalability.
It includes the following modules:

* :ref:`pmm-client` installed on every database host that you want to monitor.
  It collects server metrics, general system metrics, and |qan.name| data
  for a complete performance overview.

* :ref:`pmm-server` is the central part of |pmm| that aggregates collected data
  and presents it in the form of tables, dashboards, and graphs in a web
  interface.

The modules are packaged for easy installation and usage. It is assumed that
the user should not need to understand what are the exact tools that make up
each module and how they interact. However, if you want to leverage the full
potential of |pmm|, the internal structure is important.

.. contents::
   :local:
   :depth: 2

|pmm| is a collection of tools designed to seamlessly work together.  Some are
developed by |percona| and some are third-party open-source tools.

.. note:: The overall client-server model is not likely to change, but the set
   of tools that make up each component may evolve with the product.

The following diagram illustrates how |pmm| is currently structured:

.. image:: images/pmm-diagram.png

.. _pmm-client:

PMM Client
================================================================================

|pmm-client| packages are available for most popular |linux| distributions:

* DEB for |debian|-based distributions
  (including |ubuntu| and others)
* RPM for |red-hat.name| derivatives
  (including |centos|, |oracle-linux|, |amazon-linux|, and others)

There are also generic tarball binaries that can be used on any |linux| system.

For more information, see :ref:`install-client`.

|pmm-client| packages consist of the following:

* |pmm-admin| is a command-line tool for managing |pmm-client|,
  for example, adding and removing database instances
  that you want to monitor.
  For more information, see :ref:`pmm-admin`.

* ``pmm-mysql-queries-0`` is a service
  that manages the |qan| agent
  as it collects query performance data from |mysql|
  and sends it to the |qan| API on :ref:`pmm-server`.

* ``pmm-mongodb-queries-0`` is a service
  that manages the QAN agent
  as it collects query performance data from |mongodb|
  and sends it to |qan| API on :ref:`pmm-server`.

* ``node_exporter`` is a |prometheus| exporter
  that collects general system metrics.
  For more information, see https://github.com/percona/node_exporter.

* ``mysqld_exporter`` is a |prometheus| exporter
  that collects |mysql| server metrics.
  For more information, see https://github.com/percona/mysqld_exporter.

* ``mongodb_exporter`` is a |prometheus| exporter
  that collects |mongodb| server metrics.
  For more information, see https://github.com/percona/mongodb_exporter.

* ``proxysql_exporter`` is a |prometheus| exporter
  that collects |proxysql| performance metrics.
  For more information, see https://github.com/percona/proxysql_exporter.

.. _pmm-server:

|pmm-server|
--------------------------------------------------------------------------------

|pmm-server| runs on the machine that will be your central monitoring host.
It is distributed as an appliance via the following:

* |docker| image that you can use to run a container
* Open Virtual Appliance (OVA) that you can run in |virtualbox| or another
  hypervisor
* |ami.intro| that you can run via |aws.intro|

For more information, see :ref:`deploy-pmm.server.installing`.

|pmm-server| includes the following tools:

* |qan.intro| enables you to analyze |mysql| query performance over periods of
  time. In addition to the client-side |qan| agent, it includes the following:

  * |qan| API is the backend for storing and accessing query data collected by
    the |qan| agent running on a :ref:`pmm-client`.

  * |qan| Web App is a web application for visualizing collected |qan.name|
    data.

* |metrics-monitor| provides a historical view of metrics
  that are critical to a |mysql| or |mongodb| server instance.
  It includes the following:

  * |prometheus| is a third-party time-series database that connects to
    exporters running on a :ref:`pmm-client` and aggregates metrics collected by
    the exporters.  For more information, see `Prometheus Docs`_.

    * |consul| provides an API that a :ref:`pmm-client` can use to remotely
      list, add, and remove hosts for Prometheus.  It also stores monitoring
      metadata.  For more information, see `Consul Docs`_.

      .. warning:: Although the |consul| web UI is accessible, do not make any
         changes to the configuration.

  * |grafana| is a third-party dashboard and graph builder for visualizing data
    aggregated by |prometheus| in an intuitive web interface.  For more
    information, see `Grafana Docs`_.

    * |percona| Dashboards is a set of dashboards for |grafana| developed by
      |percona|.

* |orchestrator| is a |mysql| replication topology management
  and visualization tool.
  For more information, see: `Orchestrator Manual`_.

All tools can be accessed from the |pmm-server| web interface (landing page).
For more information, see :ref:`using`.

.. seealso::

   Default ports
      :term:`Ports` in :ref:`pmm/glossary/terminology-reference`
   Enabling orchestrator
      :term:`Orchestrator` in :ref:`pmm/glossary/terminology-reference`

.. rubric:: **References**

.. target-notes::

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/url.txt

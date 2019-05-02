.. _pmm.architecture:

--------------------------------------------------------------------------------
Client/Server architecture - an overview
--------------------------------------------------------------------------------

The |pmm| platform is based on a client-server model that enables scalability.
It includes the following modules:

* :ref:`pmm-client` installed on every database host that you want to monitor.
  It collects server metrics, general system metrics, and |query-analytics| data
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

.. image:: ../.res/graphics/png/diagram.pmm-architecture.png

.. _pmm-client:

`PMM Client <architecture.html#pmm-client>`_
================================================================================

Each |pmm-client| collects various data about general system and database
performance, and sends this data to the corresponding |pmm-server|.

The |pmm-client| package consist of the following:

* |pmm-admin| is a command-line tool for managing |pmm-client|,
  for example, adding and removing database instances
  that you want to monitor.
  For more information, see :ref:`pmm-admin`.
* ``pmm-mysql-queries-0`` is a service that manages the |qan| agent as it
  collects query performance data from |mysql| and sends it to the |qan| API on
  :ref:`pmm-server`.
* ``pmm-mongodb-queries-0`` is a service that manages the |qan| agent as it
  collects query performance data from |mongodb| and sends it to |qan| API on
  :ref:`pmm-server`.
* |node-exporter| is a |prometheus| exporter that collects general system
  metrics.
* |mysqld-exporter| is a |prometheus| exporter that collects |mysql| server
  metrics.
* |mongodb-exporter| is a |prometheus| exporter that collects |mongodb| server
  metrics.
* |proxysql-exporter| is a |prometheus| exporter that collects |proxysql|
  performance metrics.

.. seealso::

   How to install |pmm-client|
      :ref:`deploy-pmm.client.installing`

   How to pass exporter specific options when adding a monitoring service
      :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`

   List of available exporter options
      :ref:`pmm.list.exporter`

.. _pmm-server:

`PMM Server <architecture.html#pmm-server>`_
================================================================================

|pmm-server| runs on the machine that will be your central monitoring host.
It is distributed as an appliance via the following:

* |docker| image that you can use to run a container
* |abbr.ova| that you can run in |virtualbox| or another
  hypervisor
* |abbr.ami| that you can run via |amazon-web-services|

For more information, see :ref:`deploy-pmm.server.installing`.

|pmm-server| includes the following tools:

* |query-analytics| enables you to analyze |mysql| query performance over periods of
  time. In addition to the client-side |qan| agent, it includes the following:

  * |qan| API is the backend for storing and accessing query data collected by
    the |qan| agent running on a :ref:`pmm-client`.

  * |qan| Web App is a web application for visualizing collected |query-analytics|
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
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`
   Enabling orchestrator
      :ref:`Orchestrator <Orchestrator>` in :ref:`pmm.glossary.terminology-reference`

.. include:: ../.res/replace.txt

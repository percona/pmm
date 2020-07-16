.. _pmm.architecture:

--------------------------------------------------------------------------------
Client/Server Architecture - an Overview
--------------------------------------------------------------------------------

The |pmm| platform is based on a client-server model that enables scalability.
It includes the following modules:

* :ref:`pmm-client` installed on every database host that you want to monitor.
  It collects server metrics, general system metrics, and |query-analytics| data
  for a complete performance overview.

* :ref:`pmm-server` is the central part of |pmm| that aggregates collected data
  and presents it in the form of tables, dashboards, and graphs in a web
  interface.

* :ref:`pmm-platform` provides value-added services for |pmm|.

.. image:: ../.res/graphics/png/diagram.pmm.client-server-platform.png

The modules are packaged for easy installation and usage. It is assumed that
the user should not need to understand what are the exact tools that make up
each module and how they interact. However, if you want to leverage the full
potential of |pmm|, the internal structure is important.

.. contents::
   :local:
   :depth: 1

|pmm| is a collection of tools designed to seamlessly work together.  Some are
developed by |percona| and some are third-party open-source tools.

.. note:: The overall client-server model is not likely to change, but the set
   of tools that make up each component may evolve with the product.

The following sections illustrates how |pmm| is currently structured.

.. _pmm-client:

`PMM Client <architecture.html#pmm-client>`_
================================================================================

.. image:: ../.res/graphics/png/diagram.pmm.client-architecture.png

Each |pmm-client| collects various data about general system and database
performance, and sends this data to the corresponding |pmm-server|.

The |pmm-client| package consist of the following:

* |pmm-admin| is a command-line tool for managing |pmm-client|,
  for example, adding and removing database instances
  that you want to monitor.
  For more information, see :ref:`pmm.ref.pmm-admin`.
* **pmm-agent** is a client-side component a minimal command-line interface,
  which is a central entry point in charge for bringing the client
  functionality: it carries on client's authentication, gets the client
  configuration stored on the PMM Server, manages exporters and other agents.
* |node-exporter| is a |prometheus| exporter that collects general system
  metrics.
* |mysqld-exporter| is a |prometheus| exporter that collects |mysql| server
  metrics.
* |mongodb-exporter| is a |prometheus| exporter that collects |mongodb| server
  metrics.
* |postgresql-exporter| is a |prometheus| exporter that collects |postgresql|
  performance metrics.
* |proxysql-exporter| is a |prometheus| exporter that collects |proxysql|
  performance metrics.

To make data transfer from PMM Client to PMM Server secure, all exporters are
able to use SSL/TLS encrypted connections, and their communication with the PMM
server is protected by the HTTP basic authentication.

.. note:: Credentials used in communication between the exporters and the PMM
   Server are the following ones:

   * login is "pmm" 

   * password is equal to Agent ID, which can be seen e.g. on the Inventory
     Dashboard.

.. seealso::

   How to install |pmm-client|
      :ref:`deploy-pmm.client.installing`

.. _pmm-server:

`PMM Server <architecture.html#pmm-server>`_
================================================================================

.. image:: ../.res/graphics/png/diagram.pmm.server-architecture.png

|pmm-server| runs on the machine that will be your central monitoring host.
It is distributed as an appliance via the following:

* |docker| image that you can use to run a container
* |abbr.ova| that you can run in |virtualbox| or another
  hypervisor
* |abbr.ami| that you can run via |amazon-web-services|

For more information, see `Installing PMM Server <https://www.percona.com/doc/percona-monitoring-and-management/2.x/install/index-server.html>`_.

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

  * ClickHouse is a third-party column-oriented database that facilitates
    the |query-analytics| functionality. For more information, see
    `ClickHouse Docs <https://clickhouse.yandex/>`_.

  * |grafana| is a third-party dashboard and graph builder for visualizing data
    aggregated by |prometheus| in an intuitive web interface.  For more
    information, see `Grafana Docs`_.

    * |percona| Dashboards is a set of dashboards for |grafana| developed by
      |percona|.

All tools can be accessed from the |pmm-server| web interface (landing page).
For more information, see :ref:`using`.

.. _pmm-platform:

`Percona Platform <architecture.html#percona-platform>`_
================================================================================

|percona-platform| provides the following value-added services to |pmm|.

|stt|
-----------------------------------------------

|stt| checks registered database instances for a range of common security issues.
This service requires the :guilabel:`Telemetry` setting to be on.

.. seealso::

   - :ref:`Security Threat Tool main page <platform.stt>`

   - :ref:`Security Threat Tool settings <server-admin-gui-stt>`
   
.. _`Prometheus Docs`: https://prometheus.io/docs/introduction/overview/
.. _`Consul Docs`: https://www.consul.io/docs/
.. _`Grafana Docs`: http://docs.grafana.org/
.. _`Orchestrator Manual`: https://github.com/outbrain/orchestrator/wiki/Orchestrator-Manual

.. include:: ../.res/replace.txt


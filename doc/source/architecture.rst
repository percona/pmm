.. _architecture:

==============================================
Percona Monitoring and Management Architecture
==============================================

The PMM platform is based on a client-server model that enables scalability.  It
includes the following modules:

* :ref:`pmm-client` installed on every database host
  that you want to monitor.
  It collects server metrics, general system metrics,
  and query analytics data for a complete performance overview.

* :ref:`pmm-server` is the central part of PMM
  that aggregates collected data and presents it in the form of tables,
  dashboards, and graphs in a web interface.

The modules are packaged for easy installation and usage.
It is assumed that the user should not need to understand
what are the exact tools that make up each module and how they interact.
However, if you want to leverage the full potential of PMM,
internal structure is important.

.. contents::
   :local:
   :depth: 2

PMM is a collection of tools designed to seamlessly work together.
Some are developed by Percona and some are third-party open-source tools.

.. note:: The overall client-server model is not likely to change,
   but the set of tools that make up each component
   may evolve with the product.

The following diagram illustrates how PMM is currently structured:

.. image:: images/pmm-diagram.png

.. _pmm-client:

PMM Client
----------

*PMM Client* packages are available for most popular Linux distributions:

* DEB for Debian-based distributions
  (including Ubuntu and others)
* RPM for Red Hat Enterprise Linux derivatives
  (including CentOS, Oracle Linux, Amazon Linux, and others)

There are also generic tarball binaries that can be used on any Linux system.

For more information, see :ref:`install-client`.

*PMM Client* packages consist of the following:

* ``pmm-admin`` is a command-line tool for managing *PMM Client*,
  for example, adding and removing database instances
  that you want to monitor.
  For more information, see :ref:`pmm-admin`.

* ``pmm-mysql-queries-0`` is a service
  that manages the Query Analytics (QAN) agent
  as it collects query performance data from MySQL
  and sends it to QAN API on :ref:`pmm-server`.

* ``pmm-mongodb-queries-0`` is a service
  that manages the QAN agent
  as it collects query performance data from MongoDB
  and sends it to QAN API on :ref:`pmm-server`.

* ``node_exporter`` is a Prometheus exporter
  that collects general system metrics.
  For more information, see https://github.com/percona/node_exporter.

* ``mysqld_exporter`` is a Prometheus exporter
  that collects MySQL server metrics.
  For more information, see https://github.com/percona/mysqld_exporter.

* ``mongodb_exporter`` is a Prometheus exporter
  that collects MongoDB server metrics.
  For more information, see https://github.com/percona/mongodb_exporter.

* ``proxysql_exporter`` is a Prometheus exporter
  that collects ProxySQL performance metrics.
  For more information, see https://github.com/percona/proxysql_exporter.

.. _pmm-server:

PMM Server
----------

*PMM Server* runs on the machine that will be your central monitoring host.
It is distributed as an appliance via the following:

* Docker image that you can use to run a container
* Open Virtual Appliance (OVA)
  that you can run in VirtualBox or another hypervisor
* Amazon Machine Image (AMI) that you can run via Amazon Web Services (AWS)

For more information, see :ref:`deploy-pmm.server.installing`.

*PMM Server* consists of the following tools:

* **Query Analytics** (QAN) enables you to analyze
  MySQL query performance over periods of time.
  In addition to the client-side QAN agent,
  it includes the following:

  * **QAN API** is the backend for storing and accessing query data
    collected by the QAN agent running on a :ref:`pmm-client`.

  * **QAN Web App** is a web application
    for visualizing collected Query Analytics data.

* **Metrics Monitor** (MM) provides a historical view of metrics
  that are critical to a MySQL or MongoDB server instance.
  It includes the following:

  * **Prometheus** is a third-party time-series database
    that connects to exporters running on a :ref:`pmm-client`
    and aggregates metrics collected by the exporters.
    For more information, see `Prometheus Docs`_.

    .. _`Prometheus Docs`: https://prometheus.io/docs/introduction/overview/

    * **Consul** provides an API
      that a :ref:`pmm-client` can use to remotely list, add,
      and remove hosts for Prometheus.
      It also stores monitoring metadata.
      For more information, see `Consul Docs`_.

      .. warning:: Although the Consul web UI is accessible,
         do not make any changes to the configuration.

      .. _`Consul Docs`: https://www.consul.io/docs/

  * **Grafana** is a third-party dashboard and graph builder
    for visualizing data aggregated by *Prometheus*
    in an intuitive web interface.
    For more information, see `Grafana Docs`_.

    .. _`Grafana Docs`: http://docs.grafana.org/

    * **Percona Dashboards** is a set of dashboards
      for *Grafana* developed by Percona.

* **Orchestrator** is a MySQL replication topology management
  and visualization tool.
  For more information, see: `Orchestrator Manual`_.

  .. _`Orchestrator Manual`:
     https://github.com/outbrain/orchestrator/wiki/Orchestrator-Manual

All tools can be accessed from the *PMM Server* web interface (landing page).
For more information, see :ref:`using`.

.. DEPRECATED: moving deployment related information to the dedicated deployment section.
   .. _scenarios:

.. rubric:: References

.. target-notes::


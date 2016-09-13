.. _architecture:

==============================================
Percona Monitoring and Management Architecture
==============================================

The PMM platform is based on a simple client-server model
that enables efficient scalability.
It includes the following modules:

* :ref:`pmm-client` is installed on every database host
  that you want to monitor.
  It collects server metrics, general system metrics,
  and query analytics data for a complete performance overview.
  Collected data is sent to *PMM Server*.

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

*PMM Client* is distributed as a tarball
that you can install on any database host that you want to monitor.
It consists of the following:

* ``pmm-admin`` is a command-line tool for managing *PMM Client*
  , for example, adding and removing database instances
  that you want to monitor.

* ``percona-qan-agent`` is a service
  that manages the Query Analytics (QAN) agent
  as it collects query performance data.
  It also connects with QAN API in :ref:`pmm-server`
  and sends over collected data.
  The service is registered and started
  when you install the *PMM Client* package.

* ``node_exporter`` is a Prometheus exporter
  that collects general system metrics.
  For more information, see https://github.com/prometheus/node_exporter.

* ``mysqld_exporter`` is a Prometheus exporter
  that collects MySQL server metrics.
  For more information, see https://github.com/percona/mysqld_exporter.

* ``mongodb_exporter`` is a Prometheus exporter
  that collects MongoDB server metrics.
  For more information, see https://github.com/Percona-Lab/prometheus_mongodb_exporter.

.. _pmm-server:

PMM Server
----------

*PMM Server* is distributed as a Docker image
that you can use to run a container on the machine
that will be your central monitoring host.
It consists of the following tools:

* **Query Analytics** (QAN) enables you to analyze
  MySQL query performance over periods of time.
  In addition to the client-side QAN agent,
  it includes the following:

  * **QAN API** is the backend for storing and accessing query data
    collected by ``percona-qan-agent`` running on a :ref:`pmm-client`.

  * **QAN Web App** is a web application
    for visualizing collected Query Analytics data.

* **Metrics Monitor** (MM) provides a historical view of metrics
  that are critical to a MySQL or MongoDB server instance.
  It includes the following:

  * **Prometheus** is a third-party time-series database
    that connects to exporters running on a :ref:`pmm-client`
    and aggregates colleted metrics.
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

Both tools (QAN and MM) are accessed
from the *PMM Server* web interface (landing page).
For more information, see :ref:`using`.

.. _scenarios:

Deployment Scenarios
--------------------

PMM is designed to be scalable for various environments.
Depending on the size and complexity of your infrastructure,
you can deploy it in several ways.

Simple Scenario
***************

If you have just one MySQL or MongoDB server,
you can install and run both modules 
(*PMM Client* and *PMM Server*)
on this one database host.

Typical Scenario
****************

It is more typical to have several MySQL and MongoDB server instances
distributed over different hosts.
In this case, you can run *PMM Server* on a dedicated monitoring host,
and install *PMM Client* on every database host that you want to monitor.
Data from hosts will be aggregated on the PMM Server.

.. rubric:: References

.. target-notes::


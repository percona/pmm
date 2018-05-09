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

.. image:: .res/graphics/png/diagram.pmm-architecture.png

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
* |node-exporter| is a |prometheus| exporter that collects general system
  metrics.
* |mysqld-exporter| is a |prometheus| exporter that collects |mysql| server
  metrics.
* |mongodb-exporter| is a |prometheus| exporter that collects |mongodb| server
  metrics.
* |proxysql-exporter| is a |prometheus| exporter that collects |proxysql|
  performance metrics.

.. seealso::

   |mongodb-exporter| repository at |github|
       `percona/mongodb_exporter`_
   |mysqld-exporter| repository at |github|
       `percona/mysqld_exporter`_
   |node-exporter| repository at |github|
       `percona/node_exporter`_
   |proxysql-exporter| repository at |github|
       `percona/proxysql_exporter`_
   Passing exporter options when adding a monitoring service
       :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`
   List of available exporter options
       :ref:`pmm/list.exporter`

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

.. _pmm/using.orchestrator:

|orchestrator|
================================================================================

|orchestrator| is a |mysql| replication topology management and visualization
tool.  If it is enabled, you can access it using the ``/orchestrator`` URL after
|pmm-server| address.  Alternatively, you can click the
|gui.mysql-replication-topology-manager| button on the |pmm-server| landing
page.

To use it, create a |mysql| user for |orchestrator| on all managed instances:

.. include:: .res/code/sql.org
   :start-after: +grant.orc-client-user+
   :end-before: #+end-block

.. note:: The credentials in the previous example are default.
   If you use a different user name or password,
   you have to pass them when
   :ref:`running PMM Server <deploy-pmm.server.installing>`
   using the
   :term:`ORCHESTRATOR_PASSWORD <ORCHESTRATOR_PASSWORD (Option)>`
   and
   :term:`ORCHESTRATOR_USER  <ORCHESTRATOR_USER (Option)>` options.

   .. include:: .res/code/sh.org
      :start-after: +docker.run.orchestrator-enabled.orchestrator-user.orchestrator-password+
      :end-before: #+end-block

Then you can use the |gui.discover| page in the |orchestrator| web interface
to add the instances to the topology.

.. note:: **Orchestrator is not enabled by default starting with PMM 1.3.0**

   |orchestrator| was included into |pmm| for experimental purposes.  It is a
   standalone tool, not integrated with |pmm| other than that you can access it
   from the landing page.

   In version 1.3.0 and later, |orchestrator| is not enabled
   by default. To enable it, see
   :ref:`pmm/docker.additional_option` in the
   :ref:`run-server-docker` section.

.. .. rubric:: References

.. .. target-notes::

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/url.txt

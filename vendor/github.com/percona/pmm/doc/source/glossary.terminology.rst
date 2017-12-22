.. _pmm/glossary/terminology-reference:

================================================================================
Terminology Reference
================================================================================

.. glossary::
   :sorted:

   Data retention

      By default, |prometheus| stores time-series data for 30 days,
      and :term:`QAN` stores query data for 8 days.

      Depending on available disk space and your requirements, you may
      need to adjust data retention time.

      You can control data retention by passing the :term:`METRICS_RETENTION
      <METRICS_RETENTION (Option)>` and :term:`QUERIES_RETENTION
      <QUERIES_RETENTION (Option)>` environment variables when :ref:`creating
      and running the PMM Server container <server-container>`.

      .. seealso::

	 Metrics retention

	    :term:`METRICS_RETENTION <METRICS_RETENTION (Option)>`

	 Queries retention

	    :term:`QUERIES_RETENTION <QUERIES_RETENTION (Option)>`

   Default ports

      See :term:`Ports`.

   Ports

      The following ports must be open to enable communication between the
      :term:`PMM Server` and :term:`PMM clients <PMM Client>`.

      |pmm-server| should keep ports 80 or 443 ports open for computers where
      |pmm-client| is installed to access the |pmm| web interface.

      On each computer where |pmm-client| is installed, the following ports must
      be open. These are default ports that you can change when adding the
      respective monitoring service with the |pmm-admin.add| command.

      42000
         For |pmm| to collect genenal system metrics.
      42001
         This port is used by a service which collects query performance data
         and makes it available to |qan|.
      42002
         For |pmm| to collect |mysql| server metrics.
      42003
         For |pmm| to collect |mongodb| server metrics.
      42004
	 For |pmm| to collect |proxysql| server metrics.

      .. seealso::

	 Setting up a firewall on |centos|
	    https://www.digitalocean.com/community/tutorials/how-to-set-up-a-firewall-using-firewalld-on-centos-7
	 Setting up a firewall on |ubuntu|
	    https://www.digitalocean.com/community/tutorials/how-to-set-up-a-firewall-with-ufw-on-ubuntu-16-04

   Telemetry

      |percona| may collect some statistics about the machine where |pmm| is running.

      This statistics includes the following information:

      - |pmm-server| unique ID
      - |pmm| version
      - The name and version of the operating system, |ami| or virtual appliance
      - |mysql| version
      - |perl| version

      You may disable telemetry :ref:`by passing an additional parameter
      <pmm/docker.additional_parameters>` to |docker|.

      .. include:: .res/code/sh.org
	 :start-after: docker.run.disable-telemetry
	 :end-before: #+end-block

   External Monitoring Service

      A monitoring service which is not provided by :term:`PMM` directly. It is
      bound to a running |prometheus| exporter. As soon as such an service is
      added, you can set up the :term:`Metrics Monitor <Metrics Monitor (MM)>`
      to display its graphs.

   Orchestrator

      The topology manager for |mysql|. By default it is disabled for the
      :term:`PMM Server`. To enable it, set the :option:`ORCHESTRATOR_ENABLED`.

      .. seealso::

	 - :ref:`Enabling orchestrator when running a docker container <pmm/docker.additional_parameters>`.

   Data Source Name

      A database server attribute found on the :term:`QAN` page. It informs how
      :term:`PMM` connects to the selected database.

   Version

      A database server attribute found on the :term:`QAN` page. it informs the
      full version of the monitored database server, as well as the product
      name, revision and release number.
	    
   DSN

      See :term:`Data Source Name`

   Grand Total Time

      Grand Total Time.(percent of grand total time) is the percentage
      of time that the database server spent running a specific query,
      compared to the total time it spent running all queries during
      the selected period of time.

   %GTT

      See :term:`Grand Total Time`

   Query Summary Table

      A tool within :term:`QAN` which lists the queries which were run
      on the selected database server during the :term:`selected time
      or date range`.

   Query Metrics Table

      A tool within :term:`QAN` which lists metrics applicable to the query
      selected in the :term:`query summary table`.

   Selected Time or Date Range

      A predefined time period (see :term:`Quick ranges`), such as 1 hour, or a
      range of dates that :term:`QAN` uses to collects metrics.

   Quick ranges

      Predefined time periods which are used by :term:`QAN` to collect metrics
      for queries. The following quick ranges are available:

      - last hour
      - last three hours
      - last five hours
      - last twelve hours
      - last twenty four hours
      - last five days

   Query Load

      The percentage of time that the |mysql| server spent executing a specific query.

   Query Abstract

      Query pattern with placeholders. This term appears in :term:`QAN <Query
      Analytics (QAN)>` as an attribute of queries.

   Query ID

      A :term:`query fingerprint` which groups similar queries.

   Query Fingerprint

      See :term:`Query Abstract`

   PMM Version

      The version of PMM appears at the bottom of the :term:`PMM server home page <PMM Home Page>`.

      .. figure:: ./images/update-button.png

	 To update your |pmm-server|, click the *Update* button
	 located next to the version number.

      .. seealso::

	 Checking the version of |pmm-server|

	     :term:`PMM Server Version`

   PMM Docker Image

      A docker image which enables installing the |pmm-server| by
      using :program:`docker`.

      .. seealso::

	 Installing |pmm-server| using |docker|
	    :ref:`run-server-docker`

   PMM Home Page

      The starting page of the PMM portal from which you can open the tools of
      PMM, view or download documentation.

      On the PMM home page, you can also find the version number and a button to
      update your |pmm-server| (see :term:`PMM Version`).

   PMM

      Percona Monitoring and Management

   pmm-admin

      A program which changes the configuration of the :term:`PMM Client`. See
      detailed documentation in the :ref:`pmm-admin` section.

   Monitoring service

      A special service which collects information from the database instance
      where :term:`PMM Client` is installed.

      To add a monitoring service, use the :program:`pmm-admin add` command.

      .. seealso::

	 Passing parameters to a monitoring service
	    :ref:`pmm.pmm-admin.monitoring-service.pass-parameter`

   Metrics

      A series of data which are visualized in |pmm|.

   Metrics Monitor (MM)
   
      Component of :term:`PMM Server` that provides a historical view of
      :term:`metrics <Metrics>` critical to a |mysql| server instance.

   PMM Client
   
      Collects |mysql| server metrics, general system metrics,
      and query analytics data for a complete performance overview.

      The collected data is sent to :term:`PMM Server`.

      For more information, see :ref:`architecture`.

   PMM Server

      Aggregates data collected by :term:`PMM Client` and presents it in the
      form of tables, dashboards, and graphs in a web interface.

      |pmm-server| combines the backend API and storage for collected data with
      a frontend for viewing time-based graphs and performing thorough analysis
      of your |mysql| and |mongodb| hosts through a web interface.

      Run |pmm-server| on a host that you will use to access this data.

      .. seealso::

	 PMM Architecture

	    :ref:`architecture`

   Query Analytics (QAN)

      Component of :term:`PMM Server` that enables you to analyze
      |mysql| query performance over periods of time.

   PMM Server Version

      If :term:`PMM Server` is installed via |docker|, you can check
      the current |pmm-server| version by running |docker.exec|:

      |tip.run-this.root|

      .. include:: .res/code/sh.org
	 :start-after: +docker.exec.it.pmm-server.head+
	 :end-before: #+end-block

   QAN

      See :term:`Query Analytics (QAN)`

   Query Metrics Summary Table

      An element of :term:`Query Analytics (QAN)` which displays the available
      metrics for the selected query.
   
.. include:: .res/replace/name.txt
.. include:: .res/replace/option.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/fragment.txt

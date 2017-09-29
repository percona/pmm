==========
 Glossary
==========

.. glossary::
   :sorted:

   Orchestrator

      The topology manager for MySQL. By default it is disabled for the
      :term:`PMM Server`. To enable it, set the :option:`ORCHESTRATOR_ENABLED`.

      .. seealso::

	 - :ref:`Enabling orchestrator when running a docker container <pmm/docker.additional_parameters>`.

   ORCHESTRATOR_ENABLED (Option)

      See :term:`Orchestrator`

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

      The percentage of time that the MySQL server spent executing a specific query.

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

	 To update your |product-abbrev| server, click the *Update* button
	 located next to the version number.

   PMM Docker Image

      A docker image which enables installing the |product-abbrev| server by
      using :program:`docker`.

      For more information about how to install |product-abbrev| server using
      this option, see :ref:`run-server-docker`.

   PMM Home Page

      The starting page of the PMM portal from which you can open the tools of
      PMM, view or download documentation.

      On the PMM home page, you can also find the version number and a button to
      update your PMM server (see :term:`PMM Version`).

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

	 - :ref:`Passing parameters to a monitoring service <pmm.pmm-admin.monitoring-service.pass-parameter>`

   Metrics

      A series of data which are visualized in PMM.

   Metrics Monitor (MM)
   
      Component of :term:`PMM Server` that provides a historical view of
      :term:`metrics <Metrics>` critical to a MySQL server instance.

   PMM Client
   
      Collects MySQL server metrics, general system metrics,
      and query analytics data for a complete performance overview.

      The collected data is sent to :term:`PMM Server`.

      For more information, see :ref:`architecture`.

   PMM Server

      Aggregates data collected by :term:`PMM Client` and presents it in the
      form of tables, dashboards, and graphs in a web interface.

      *PMM Server* combines the backend API and storage for collected data with
      a frontend for viewing time-based graphs and performing thorough analysis
      of your MySQL and MongoDB hosts through a web interface.

      Run |product-abbrev| server on a host that you will use to access this data.


      For more information, see :ref:`architecture`.

   Query Analytics (QAN)

      Component of :term:`PMM Server` that enables you to analyze
      MySQL query performance over periods of time.

   QAN

      See :term:`Query Analytics (QAN)`

   Query Metrics Summary Table

      An element of :term:`Query Analytics (QAN)` which displays the available
      metrics for the selected query.
   
.. include:: replace.txt

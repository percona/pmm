==========
 Glossary
==========

.. glossary::
   :sorted:

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

.. include:: replace.txt

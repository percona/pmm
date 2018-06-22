.. _dashboard.prometheus:

Prometheus Dashboard
================================================================================

The |prometheus| dashboard informs how |prometheus| functions. 

.. seealso::

   Overview of |pmm|
      :ref:`using`
   All dashboards
      :ref:`pmm.dashboard.list`

|prometheus| overview
--------------------------------------------------------------------------------

This section shows the most essential parameters of the system where
|prometheus| is running, such as CPU and memory usage, scrapes performed or the
amount of memory that time series consume.

Resources
--------------------------------------------------------------------------------

This section provides details about the consumption of CPU and memory by the
|prometheus| process.

Storage
--------------------------------------------------------------------------------

This section includes a collection of metrics related to the usage of storage. It includes the following metrics:

.. hlist::
   :columns: 2

   - Time Series
   - Local Storage Urgency Score
   - |prometheus| Problems
   - |prometheus| Checkpointing
   - |prometheus| Out of Order Samples
   - Chunks Memory Details
   - Series Operations
   - |prometheus| Storage Operations
   - ChunkDesc Memory Details

  

Scraping
--------------------------------------------------------------------------------

This section contains metrics that help monitor the scraping process. This
section contains the following metrics:

.. hlist::
   :columns: 2

   - Ingestion
   - |Prometheus| Targets
   - Scraped Target by Job
   - Scrape Time by Job
   - Scraped Target by Instance
   - Scraped Time by Instance
   - Scrapes by Target Frequency
   - Scrape Frequency Versus Target
   - Scraping Time Drift
   - |prometheus| Scrape Interval Variance
   - Slowest Jobs
   - Largest Samples Jobs

Queries
--------------------------------------------------------------------------------

This section contains metrics that monitor |prometheus| queries. This section contains the following metrics:

.. hlist::
   :columns: 2

   - |prometheus| Queries
   - |prometheus| Query Execution
   - |prometheus| Query Execution Latency
   - |prometheus| Query Execution Load

Network
--------------------------------------------------------------------------------

Metrics in this section help detect network problems.

.. hlist::
   :columns: 2

   - HTTP Requests by Handler
   - HTTP Errors
   - HTTP Avg Response time by Handler
   - HTTP 99% Percentile Response time by Handler
   - HTTP Response Average Size by Handler
   - HTTP 99% Percentile Response Size

Time Series Information
--------------------------------------------------------------------------------

This section shows the top 10 metrics by time series count and the top 10 hosts
by time series count.

System Level Metrics
--------------------------------------------------------------------------------

Metrics in this section give an overview of the essential system characteristics
of |pmm-server|. This information is also available from the |system-overview|
dashboard.

|pmm| Server Logs
--------------------------------------------------------------------------------

This section contains a link to download the logs collected from your
|pmm-server| and further analyze possible problems. The exported logs are
requested when you submit a bug report.

.. include:: .res/replace/name.txt
	

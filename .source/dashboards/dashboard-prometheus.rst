.. _dashboard-prometheus:

##########
Prometheus
##########

.. image:: /_images/PMM_Prometheus.jpg

*******************
Prometheus Overview
*******************

This section shows the most essential parameters of the system where Prometheus is running, such as CPU and memory usage, scrapes performed and the samples ingested in the head block.

*********
Resources
*********

This section provides details about the consumption of CPU and memory by the Prometheus process. This section contains the following metrics:

- Prometheus Process CPU Usage
- Prometheus Process Memory Usage
- Disk Space Utilization

***********************
Storage (TSDB) Overview
***********************

This section includes a collection of metrics related to the usage of storage. It includes the following metrics:

- Data blocks (Number of currently loaded data blocks)
- Total chunks in the head block
- Number of series in the head block
- Current retention period of the head block
- Activity with chunks in the head block
- Reload block data from disk

********
Scraping
********

This section contains metrics that help monitor the scraping process. This section contains the following metrics:

- Ingestion
- Prometheus Targets
- Scraped Target by Job
- Scrape Time by Job
- Scraped Target by Instance
- Scraped Time by Instance
- Scrapes by Target Frequency
- Scrape Frequency Versus Target
- Scraping Time Drift
- Prometheus Scrape Interval Variance
- Slowest Jobs
- Largest Samples Jobs

*******
Queries
*******

This section contains metrics that monitor Prometheus queries. This section contains the following metrics:

- Prometheus Queries
- Prometheus Query Execution
- Prometheus Query Execution Latency
- Prometheus Query Execution Load

*******
Network
*******

Metrics in this section help detect network problems.

- HTTP Requests by Handler
- HTTP Errors
- HTTP Avg Response time by Handler
- HTTP 99% Percentile Response time by Handler
- HTTP Response Average Size by Handler
- HTTP 99% Percentile Response Size

***********************
Time Series Information
***********************

This section shows the top 10 metrics by time series count and the top 10 hosts by time series count.

********************
System Level Metrics
********************

Metrics in this section give an overview of the essential system characteristics of PMM Server. This information is also available from the :ref:`dashboard-nodes-overview` dashboard.

***************
PMM Server Logs
***************

This section contains a link to download the logs collected from your PMM Server and further analyze possible problems. The exported logs are requested when you submit a bug report.


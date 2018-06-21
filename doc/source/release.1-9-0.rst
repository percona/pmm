.. _pmm/release/1-9-0:

|pmm.name| |release|
********************************************************************************

:Date: April 4, 2018

For more information about this release, see the `release announcement`_.

.. contents::
   :local:
      
|status.improved|: CloudWatch RDS metrics
================================================================================

:JIRA Ticket ID: :pmmbug:`2010`

6 node-specific dashboards now display |amazon-rds| node-level metrics:

- Cross_Server (Network Traffic)
- Disk Performance (Disk Latency)
- Home Dashboard (Network IO)
- MySQL Overview (Disk Latency, Network traffic)
- Summary Dashboard (Network Traffic)
- System Overview (Network Traffic)

|status.improved|: AWS Add Instance changes
================================================================================

:JIRA Ticket ID: :pmmbug:`1823`

The |aws| add instance interface has changed to provide more information about
how to add an |amazon-aurora| |mysql| or |amazon-rds| |mysql| instance.

|status.improved|: |aws| Settings in Documentation
================================================================================

:JIRA Ticket ID: :pmmbug:`1788`

The documentation highlights connectivity best practices, and
authentication options - IAM Role or IAM User Access Key.

|status.improved|: Low RAM Support
================================================================================

:JIRA Ticket ID: :pmmbug:`2217`

You can now run PMM Server on instances with memory as low as 512MB RAM, which
means you can deploy to  the free tier of many cloud providers if you want to
experiment with PMM.  Our memory calculation is now:

.. code-block:: bash

    METRICS_MEMORY_MULTIPLIED=$(( (${MEMORY_AVAIABLE} - 256*1024*1024) / 100 * 40 ))
    if [[ $METRICS_MEMORY_MULTIPLIED < $((128*1024*1024)) ]]; then
        METRICS_MEMORY_MULTIPLIED=$((128*1024*1024))
    fi

|status.new|: |percona| Snapshot Server
================================================================================    

:JIRA Ticket ID: :pmmbug:`2058`

The button for sharing to the |grafana| publicly hosted platform has been
replaced and now directs to the host administered by |percona|. Your dashboard
will be written to |percona| snapshots and only |percona| engineers will be able
to retrieve the data.

Snapshots automatically expire after 90 days; when sharing, you can configure a
shorter retention period.

|status.new|: Export of |pmm-server| Logs
================================================================================

:JIRA Ticket ID: :pmmbug:`1274`

The logs from |pmm-server| can be exported using single button-click, avoiding
the need to log in manually to the docker container.

|status.improved|: Faster Loading of the Index Page
================================================================================

:JIRA Ticket ID: :pmmbug:`2215`

The load time of the index page has been improved with |gzip| and |http2|
enabled.

|h.bug-fix-releases|
================================================================================

|tip.bug-fix-release-list| |release|:

- :ref:`pmm/release/1-9-1`

.. seealso::

   All releases
      :ref:`pmm/release/list`

   Latest release
      :ref:`pmm/release/latest`

|h.issues|
================================================================================

Release |release| of |pmm.name| contains new features, improvements, and bug
fixes registered in the following |jira| tickets:

.. rubric:: |h.new-features|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`781`
     - Plot new PXC 5.7.17, 5.7.18 status variables on new graphs for PXC Galera, PXC Overview dashboards
   * - :pmmbug:`1274`
     - Export |pmm-server| logs as zip file to the browser
   * - :pmmbug:`2058`
     - |percona| Snapshot Server

.. rubric:: |h.improvements|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1587`
     - Use :option:`mongodb_up` variable for the |mongodb| Overview dashboard to identify if a host is |mongodb|.
   * - :pmmbug:`1788`
     - |aws| Credentials form changes
   * - :pmmbug:`1823`
     - |aws| Install wizard improvements
   * - :pmmbug:`2010`
     - System dashboards update to be compatible with RDS nodes
   * - :pmmbug:`2118`
     - Update grafana config for metric series that will not go above 1.0
   * - :pmmbug:`2215`
     - |pmm| Web speed improvements
   * - :pmmbug:`2216`
     - |pmm| can now be started on systems without memory limit capabilities in the kernel
   * - :pmmbug:`2217`
     - |pmm-server| can now run in |docker| with 512 Mb memory
   * - :pmmbug:`2252`
     - Better handling of variables in the navigation menu

.. rubric:: |h.bug-fixes|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`605`
     - :program:`pt-mysql-summary` requires additional configuration
   * - :pmmbug:`941`
     - ParseSocketFromNetstat finds an incorrect socket
   * - :pmmbug:`948`
     - Wrong load reported by |qan| due to mis-alignment of time intervals
   * - :pmmbug:`1486`
     - |mysql| passwords containing the dollar sign ($) were not processed properly.
   * - :pmmbug:`1905`
     - In |qan|, the Explain command could fail in some cases.
   * - :pmmbug:`2090`
     - Minor formatting issues in |qan|
   * - :pmmbug:`2214`
     - Setting Send real query examples for Query Analytic OFF still shows the real query in example.
   * - :pmmbug:`2221`
     - no Rate of Scrapes for |mysql| & |mysql| Errors
   * - :pmmbug:`2224`
     - Exporter CPU Usage glitches 
   * - :pmmbug:`2227`
     - Auto Refresh for dashboards 
   * - :pmmbug:`2243`
     - Long host names in |grafana| dashboards are not displayed correctly
   * - :pmmbug:`2257`
     - PXC/galera cluster overview Flow control paused time has a percentage glitch 
   * - :pmmbug:`2282`
     - No data is displayed on dashboards for OVA images
   * - :pmmbug:`2296`
     - The |opt.mysql-metrics| service will not start on Ubuntu LTS 16.04
  
.. |release| replace:: 1.9.0
		       
.. _`release announcement`: https://www.percona.com/blog/2018/04/20/percona-monitoring-and-management-1-9-0-is-now-available/

.. include:: .res/replace.txt

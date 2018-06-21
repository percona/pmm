.. _pmm/release/1-6-0:

|pmm.name| |release|
********************************************************************************

:Date: January 18, 2018

For more information about this release, see the `release announcement`_.

In this release, |pmm| |grafana| metrics are made available in the
|dbd.advanced-data-exploration| dashboard. The integration with |myrocks| has
been improved and its data are now collected from |sql.show-global-status|.

The |mongodb| exporter now features two new metrics: **mongodb_up** to inform if
the |mongodb| Server is running and **mongodb_scrape_errors_total** reporting
the total number of errors when scaping |mongodb|.

The performance of the |opt.mongodb-metrics| monitoring service has been greatly
improved.

|pmm| |release| also includes version 4.6.3 of |grafana|:

- Alert list: Now shows alert state changes even after adding manual annotations
  on dashboard `#9951 <https://github.com/grafana/grafana/issues/9951>`_
- Alerting: Fixes bug where rules evaluated as firing when all conditions was
  false and using OR operator. `#9318
  <https://github.com/grafana/grafana/issues/9318>`_

|h.bug-fix-releases|
================================================================================

|tip.bug-fix-release-list| |release|:

- :ref:`pmm/release/1-6-1`

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
   * - :pmmbug:`1773`
     - |pmm| |grafana| specific metrics have been added to the
       |dbd.advanced-data-exploration| dashboard.

.. rubric:: |h.improvements|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1485`
     - Updated |myrocks| integration: |myrocks| data is now collected entirely
       from |sql.show-slave-status|; |sql.show-engine-rocksdb-status| is not
       a data source in **mysqld_exporter**.
   * - :pmmbug:`1895`
     - Update |grafana| to version 4.6.3
   * - :pmmbug:`1586`
     - The **mongodb_exporter** exporter exposes two new metrics:
        **mongodb_up** informing if the |mongodb| server is running and
        **mongodb_scrape_errors_total** informing the total number of times an
        error occurred when scraping |mongodb|.
   * - :pmmbug:`1764`
     - Various small **mongodb_exporter** improvements
   * - :pmmbug:`1942`
     - Improved the consistency of using labels in all |prometheus| related dashboards.
   * - :pmmbug:`1936`
     - Updated the |prometheus| dashboard in |metrics-monitor|
   * - :pmmbug:`1937`
     - Added the *CPU Utilization Details (Cores)* dashboard to |metrics-monitor|.
   * - :pmmbug:`1887`
     - Improved the help text for |pmm-admin| to provide more information about exporter options.
   * - :pmmbug:`1939`
     - In |metrics-monitor|, two new dashboards have been added. The
       |dbd.prometheus-exporters-overview| dashboard provides a summary of how
       exporters utilize system resources and the |dbd.prometheus-exporter-status|
       dashboard tracks the performance of each |prometheus| exporter.

.. rubric:: |h.bug-fixes|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1549`
     - Broken default auth db for |opt.mongodb-queries|
   * - :pmmbug:`1631`
     - In some cases percentage values were displayed incorrectedly for |mongodb| hosts.
   * - :pmmbug:`1640`
     - RDS exporter: simplify configuration.
   * - :pmmbug:`1760`
     - After the mongodb:metrics monitoring service was added, the usage of CPU considerably increased in |qan| versions 1.4.1 through 1.5.3.
   * - :pmmbug:`1815`
     - |qan| could show data for a |mysql| host when a |mongodb| host was selected.
   * - :pmmbug:`1888`
     - In |qan|, query metrics were not loaded when the |qan| page was refreshed.
   * - :pmmbug:`1898`
     - In |qan|, the *Per Query Stats* graph displayed incorrect values for |mongodb|
   * - :pmmbug:`1796`
     - In |metrics-monitor|, The *Top Process States Hourly* graph from the the
       *MySQL Overview* dashboard showed incorrect data.
   * - :pmmbug:`1777`
     - In |qan|, the |gui.load| column could display incorrect data.
   * - :pmmbug:`1744`
     - The error *Please provide AWS access credentials error* appeared although
       the provided credentials could be processed successfully.
   * - :pmmbug:`1676`
     - In preparation for migration to |prometheus| 2.0 we have updated the
       *System Overview* dashboard for compatibility.
   * - :pmmbug:`1920`
     - Some standard |mysql| metrics were missing from the **mysqld_exporter**
       |prometheus| exporter.
   * - :pmmbug:`1932`
     - The *Response Length* metric was not displayed for |mongodb| hosts in |qan|.

.. _`release announcement`: https://www.percona.com/blog/2018/01/19/percona-monitoring-and-management-1-6-0-is-now-available/
  
.. |release| replace:: 1.6.0
		       
.. include:: .res/replace.txt

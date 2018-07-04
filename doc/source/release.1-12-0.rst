.. _pmm/release/latest:
.. _1.12.0:

|pmm.name| |release|
********************************************************************************

:Date: June 27, 2018

For more information about this release, see the `release announcement`_.

.. contents::
   :local:

|status.improved|: Visual Explain in |qan.name|
================================================================================

:JIRA Ticket ID: :pmmbug:`2519`

In this release, |qan.name| introduces *visual explain* based on
`pt-visual-explain
<https://www.percona.com/doc/percona-toolkit/LATEST/pt-visual-explain.html>`_. This
functionality represents the |mysql| EXPLAIN output of a query plan as a left-deep
tree which is similar to how the query plan is represented inside |mysql|.

.. seealso::

   Ways to represent the |sql.explain| output in |qan.name|
      :ref:`pmm.qan.explain-section`

|status.new| dashboard: |dbd.mysql-innodb-compression|
================================================================================

:JIRA Ticket ID: :pmmbug:`2019`

In this release, PMM introduces the |dbd.mysql-innodb-compression| dashboard to
help you understand the most important characteristics of |innodb|\'s
compression.

.. seealso::

   Metrics of the |dbd.mysql-innodb-compression| dashboard
      :ref:`dashboard.mysql-innodb-compression`

|status.new| dashboard: |dbd.mysql-command-handler-counters-compare|
================================================================================

The new |dbd.mysql-command-handler-counters-compare| dashboard lets you do a
side-by-side comparison of Command (Com\_\*) and Handler statistics.

You may find this dashboard useful as a way to compare servers that share a
similar workload, for example across |mysql| instances in a pool of replicated
slaves.

.. seealso::

   Metrics of the |dbd.mysql-command-handler-counters-compare| dashboard
      :ref:`dashboard.mysql-command-handler-counters-compare`

|h.issues|
================================================================================

.. rubric:: |h.new-features-and-improvements|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`2519`
     - Display Visual Explain in Query Analytics 
   * - :pmmbug:`2019`
     - Add new Dashboard |innodb| Compression metrics 
   * - :pmmbug:`2154`
     - Add new Dashboard Compare Commands and Handler statistics 
   * - :pmmbug:`2530`
     - Add timeout flags to mongodb_exporter (thank you `unguiculus <https://github.com/unguiculus>`_ for your contribution!)
   * - :pmmbug:`2569`
     - Update the |mysql| Golang driver for |mysql| 8 compatibility 
   * - :pmmbug:`2561`
     - Update to Grafana 5.1.3 
   * - :pmmbug:`2465`
     - Improve pmm-admin debug output 
   * - :pmmbug:`2520`
     - Explain Missing Charts from |mysql| Dashboards 
   * - :pmmbug:`2119`
     - Improve Query Analytics messaging when Host = All is passed  
   * - :pmmbug:`1956`
     - Implement connection checking in mongodb_exporter 
    
.. rubric:: |h.bug-fixes|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1704`
     - Unable to connect to AtlasDB MongoDB
   * - :pmmbug:`1950`
     - pmm-admin (mongodb:metrics) doesn\'t work well with SSL secured mongodb server
   * - :pmmbug:`2134`
     - rds_exporter exports memory in Kb with node_exporter labels which are in bytes
   * - :pmmbug:`2157`
     - Cannot connect to MongoDB using URI style 
   * - :pmmbug:`2175`
     - Grafana singlestat doesn't use consistent colour when unit is of type Time 
   * - :pmmbug:`2474`
     - Data resolution on Dashboards became 15sec interval instead of 1sec 
   * - :pmmbug:`2581`
     - Improve Travis CI tests by addressing pmm-admin check-network Time Drift
   * - :pmmbug:`2582`
     - Unable to scroll on "_PMM Add Instance" page when many RDS instances exist in an AWS account 
   * - :pmmbug:`2596`
     - Set fixed height for panel content in PMM Add Instances 
   * - :pmmbug:`2600`
     - |innodb| Checkpoint Age does not show data for |mysql| 
   * - :pmmbug:`2620`
     - Fix balancerIsEnabled & balancerChunksBalanced values
   * - :pmmbug:`2634`
     - pmm-admin cannot create user for |mysql| 8 
   * - :pmmbug:`2635`
     - Improve error message while adding metrics beyond "exit status 1"

.. rubric:: Known Issues

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`2639`
     - mysql:metrics does not work on Ubuntu 18.04 

.. seealso::

   All releases
      :ref:`pmm/release/list`


.. _`release announcement`: https://www.percona.com/blog/2018/06/27/percona-monitoring-and-management-1-12-0-is-now-available/

.. |release| replace:: 1.11.0

.. include:: .res/replace.txt

.. _pmm.release.1-7-0:

|pmm.name| |release|
********************************************************************************

:Date: January 31, 2018

For more information about this release, see the `release announcement`_.

This release features improved support for external services, which enables a
|pmm-server| to store and display metrics for any available |prometheus|
exporter. 

Issues in this release
================================================================================

Release |release| of |pmm.name| contains new features, improvements, and bug
fixes registered in the following |jira| tickets:

.. rubric:: New Features

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1949`
     - New dashboard: |dbd.mysql-amazon-aurora-metrics|.

.. rubric:: Improvements

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1712`
     - Improve external exporters to add data
       monitoring from an arbitrary |prometheus| exporter running on your host.
   * - :pmmbug:`1510`
     - Rename *swap in* and *swap out* labels to *Swap In (Reads)* and *Swap Out (Writes)* accordingly.
   * - :pmmbug:`1966`
     - Remove |grafana| from the list of exporters on dashboard to
       eliminate confusion with existing |grafana| in the list of
       exporters on the current version of the dashboard.
   * - :pmmbug:`1974`
     - Add the *mongodb_up* graph in the Exporter Status dashboard to
       maintain consistency of information about exporters. This is
       done based on new metrics implemented in :pmmbug:`1586`.

.. rubric:: Bug fixes

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1967`
     - Inconsistent formulas in |prometheus| dashboards.
   * - :pmmbug:`1986`
     - Signing out with HTTP auth enabled leaves the browser *signed in*.
  
.. seealso::

   All releases
      :ref:`pmm/release/list`

.. _`release announcement`: https://www.percona.com/blog/2018/01/31/percona-monitoring-and-management-pmm-1-7-0-now-available/

.. |release| replace:: 1.7.0
		       
.. include:: .res/replace.txt

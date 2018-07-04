.. _pmm/release/1-10-0:

|pmm.name| |release|
********************************************************************************

:Date: April 20, 2018

For more information about this release, see the `release announcement`_.

.. contents::
   :local:

|status.improved|:  Annotations - display application events
================================================================================

:JIRA Ticket ID: :pmmbug:`2330`

|pmm| now supports receiving application events and displays them as |grafana|
annotations using the new command |pmm-admin.annotate|.

.. seealso::

   How to use annotations in |pmm|?
      :ref:`pmm.metrics-monitor.annotation.application-event.marking`

   |grafana| Documentation: Annotations
      http://docs.grafana.org/reference/annotations/

|status.new|: |grafana| 5.0 - Upgraded to improve the presentation of graphs
================================================================================

:JIRA Ticket ID: :pmmbug:`2332`

|grafana| 5.0 is no longer bound by panel constraints to keep all objects at the
same fixed height.  This improvement indirectly addresses the visualization
error in |pmm-server| where some graphs would appear to be on two lines.

.. seealso::

   |grafana| Documentation: What\'s new in version 5.0?
      http://docs.grafana.org/guides/whats-new-in-v5/

|status.fixed|: Switching between dashboards while maintaining the same host
================================================================================

:JIRA Ticket ID: :pmmbug:`2371`

The selected server will not change when you switch from one dashboard
to another.

|status.new|: PXC Galera replication latency graphs: compare latency across all members in a cluster
====================================================================================================

:JIRA Ticket ID: :pmmbug:`2293`

Compare latency across all members in a cluster on the
|dbd.pxc-galera-cluster-overview| dashboard.

.. seealso::
	      
   |dbd.pxc-galera-cluster-overview| dashboard
      :ref:`dashboard.pxc-galera-cluster-overview`

|h.issues|
================================================================================

Release |release| of |pmm.name| contains new features, improvements, and bug
fixes registered in the following |jira| tickets:

.. rubric:: |h.new-features-and-improvements|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`2293`
     - Add the *Galera Replication Latency* graph to the |dbd.pxc-galera-cluster-overview| dashboard.
   * - :pmmbug:`2295`
     - Improve colour selection on the |dbd.pxc-galera-cluster-overview| dashboard
   * - :pmmbug:`2330`
     - Application Annotations
   * - :pmmbug:`2332`
     - Grafana 5 update

.. rubric:: |h.bug-fixes|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`2311`
     - Fix mis-alignment in Query Analytics Metrics table
   * - :pmmbug:`2341`
     - Typo in text on password page of OVF
   * - :pmmbug:`2359`
     - Trim leading and trailing whitespaces for all fields on AWS/OVF Installation wizard
   * - :pmmbug:`2360`
     - Include a *What's new?* link for Update widget
   * - :pmmbug:`2346`
     - Arithmetic on InnoDB AHI Graphs are invalid
   * - :pmmbug:`2364`
     - QPS are wrong in QAN
   * - :pmmbug:`2388`
     - Query Analytics does not render fingerprint section in some cases
   * - :pmmbug:`2371`
     - Pass host when switching between Dashboards

.. seealso::

   All releases
      :ref:`pmm/release/list`

   Latest release
      :ref:`pmm/release/latest`

.. |release| replace:: 1.10.0

.. _`release announcement`: https://www.percona.com/blog/2018/04/20/percona-monitoring-and-management-pmm-1-10-0-is-now-available/
		       
.. include:: .res/replace.txt

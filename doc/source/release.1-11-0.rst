.. _pmm/release/1-11-0:

|pmm.name| |release|
********************************************************************************

:Date: May 23, 2018

For more information about this release, see the `release announcement`_.

.. contents::
   :local:

|status.improved|: |mysql| |slow-log| improvements
================================================================================

:JIRA Ticket ID: :pmmbug:`2432`

Enable the |slow-log| rotation to keep a limited number of |slow-log| files on
disk. By default, |pmm| keeps one |slow-log| file. The |slow-log| rotation
feature is enabled by default.

Disable the |slow-log| rotation feature if you prefer another tool, such as
|logrotate|.

.. seealso::

   How to enable monitoring of |mysql| queries with options?
      :ref:`pmm-admin.add-mysql-queries`

|status.fixed|: Predictable graphs
================================================================================

:JIRA Ticket ID: :pmmbug:`1187`

The logic of the following dashboards has been improved to better handle
predictability and also to allow zooming to look at shorter time ranges:

- |dbd.home| Dashboard
- |dbd.pxc-galera-graphs| Dashboard
- |dbd.mysql-overview| Dashboard
- |dbd.mysql-innodb-metrics| Dashboard

|status.fixed|: MySQL Exporter parsing of |my.cnf|
================================================================================

:JIRA Ticket ID: :pmmbug:`2469`

|mysqld-exporter| could ignore options without values and read a wrong section
of the the |my.cnf| file.  In this release, the parsing engine is more |mysql|
compatible.

.. seealso::

   How |pmm| uses the |mysqld-exporter|?
      :ref:`pmm-client`

|status.improved|: Annotation improvements
================================================================================

:JIRA Ticket ID: :pmmbug:`2515`

Passing multiple arguments to the |pmm-admin.annotate| command produced an
error. In this release, the parsing of arguments has changed and
multiple words supplied to the |pmm-admin.annotate| command are concatenated to
form the text of one annotation.

.. seealso::

   How to add an annotation?
      :ref:`pmm-admin/annotate`

   How to use annotations in |pmm|?
      :ref:`pmm.metrics-monitor.annotation.application-event.marking`

   |grafana| Documentation: Annotations
      http://docs.grafana.org/reference/annotations/

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
   * - :pmmbug:`2432`
     - Configurable |mysql| |slow-log| file rotation

.. rubric:: |h.bug-fixes|

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - JIRA Ticket ID
     - Description
   * - :pmmbug:`1187`
     - Graphs breaks at tight resolution
   * - :pmmbug:`2362`
     - Explain is a part of query
   * - :pmmbug:`2399`
     - RPM for PMM Server is missing some files
   * - :pmmbug:`2407`
     - Menu items are not visible on PMM QAN dashboard
   * - :pmmbug:`2469`
     - Parsing of a valid my.cnf can break the mysqld_exporter
   * - :pmmbug:`2479`
     - PXC/Galera Cluster Overview dashboard: typo in metric names
   * - :pmmbug:`2484`
     - PXC/Galera Graphs display unpredictable results each time they are refreshed
   * - :pmmbug:`2503`
     - Wrong Innodb Adaptive Hash Index Statistics
   * - :pmmbug:`2513`
     - QAN-agent always changes ``max_slowlog_size`` to **0**
   * - :pmmbug:`2514`
     - ``pmm-admin annotate help`` - fix typos
   * - :pmmbug:`2515`
     - ``pmm-admin annotate`` - more than 1 annotation

.. seealso::

   All releases
      :ref:`pmm/release/list`

   Latest release
      :ref:`pmm/release/latest`

.. _`release announcement`: https://www.percona.com/blog/2018/05/23/percona-monitoring-and-management-1-11-0-is-now-available/

.. |release| replace:: 1.11.0

.. include:: .res/replace.txt


.. _pmm/release/1-3-0:

|pmm.name| |release|
********************************************************************************

:Date: September 26, 2017

For install and upgrade instructions, see :ref:`deploy-pmm`.

In release |release|, |percona-monitoring-management| introduces a basic support
for the |myrocks| storage engine. There is a special dashboard in
|metrics-monitor| where the essential metrics of MyRocks are presented as
separate graphs. And |metrics-monitor| graphs now feature on-demand
descriptions.

There are many improvements to |abbr.qan| both in the user interface design and
in its capabilities. In this release, |qan| starts supporting all types of
MongoDB queries.

:program:`Orchestrator` is not enabled by default because leaving it in a
non-configured state was confusing to users. :ref:`It is still possible to
enable it <pmm/docker.additional_option>`.

.. rubric:: New Features


* :pmmbug:`1290`: Basic support for the metrics of the MyRocks storage engine in MySQL via the mysqld-exporter.
* :pmmbug:`1312`: Metrics Monitor now features a MyRocks dashboard.
* :pmmbug:`1330`: Basic telemetry data are collected from PMM Servers.
* :pmmbug:`1417`: A new dashboard titled *Advanced Data Exploration Dashboard* in Metrics Monitor enables exploring any data in Prometheus
* :pmmbug:`1437`: :program:`pmm-admin` allows passing parameters to exporters.
* :pmmbug:`685`:  The EXPLAIN command is now supported for MongoDB queries in |qan|.

.. rubric:: Improvements

* :pmmbug:`1262`: The system checks for updates much faster
* :pmmbug:`1015`: |qan| should shows all collections from a mongod instance. Make sure that profiling is enabled in MongoDB.
* :pmmbug:`1057`: |qan| supports all MongoDB query types.
* :pmmbug:`1270`: In |metrics-monitor|, the MariaDB dashboard host filter now displays only the hosts running MariaDB.
* :pmmbug:`1287`: The *mongodb:queries* monitoring service is not considered to be experimental any more.
  The :option:`dev-enable` option is no longer needed when you run the :program:`pmm-admin add` command to add it.
* :pmmbug:`1446`: In |metrics-monitor|, the *MySQL Active Threads* graph displays data more accurately.
* :pmmbug:`1455`: In |metrics-monitor|, features improved descriptions of the ``InnoDB Tansactions`` graph.
* :pmmbug:`1476`: In |qan|, the new interface is now useed by default.
* :pmmbug:`1479`: It is now possible to go to |qan| directly from |metrics-monitor|.
* :pmmbug:`515`: :program:`Orchestrator` is disabled by default. It is possible to enable it when running your docker container.

.. rubric:: Bug fixes

* :pmmbug:`1298`: In |qan|, the query abstract could be empty for MySQL hosts for low ranking queries. This bug is fixed to contain *Low Ranking Queries* as the value of the query abstract.
* :pmmbug:`1314`: The selected time range in |qan| could be applied incorrectly.
  This problem is not observed in the new design of |qan|.
* :pmmbug:`1398`: The |prometheus| server was not restarted after PMM was upgraded. This bug is now fixed.
* :pmmbug:`1427`: The *CPU Usage/Load* graph in the *MySQL Overview* dashboard was displayed with slightly incorrect dimensions. This bug is now solved.
* :pmmbug:`1439`: If the EXPLAIN command was not supported for the selected query, there could appear a JavaScript error.
* :pmmbug:`1472`: In some cases, the monitoring of queries for MongoDB with replication could not be enabled.
* :pmmbug:`943`: InnoDB AHI Usage Graph had incorrect naming and hit ratio computation.

  Other bug fixes in this release: :pmmbug:`1479`

.. |release| replace:: 1.3.0

.. include:: .res/replace.txt

.. _pmm/release/1-11-0:

|pmm.name| |release|
********************************************************************************

:Date: May 23, 2018

For more information about this release, see the `release announcement`_.

Issues in this release
================================================================================

.. rubric:: New Features & Improvements

- :pmmbug:`2432`: Configurable MySQL Slow Log File Rotation

.. rubric:: Bug fixes

- :pmmbug:`1187`: Graphs breaks at tight resolution
- :pmmbug:`2362`: Explain is a part of query
- :pmmbug:`2399`: RPM for PMM Server is missing some files
- :pmmbug:`2407`: Menu items are not visible on PMM QAN dashboard
- :pmmbug:`2469`: Parsing of a valid my.cnf can break the mysqld_exporter
- :pmmbug:`2479`: PXC/Galera Cluster Overview dashboard: typo in metric names
- :pmmbug:`2484`: PXC/Galera Graphs display unpredictable results each time they are refreshed
- :pmmbug:`2503`: Wrong Innodb Adaptive Hash Index Statistics
- :pmmbug:`2513`: QAN-agent always changes ``max_slowlog_size`` to **0**
- :pmmbug:`2514`: ``pmm-admin annotate help`` - fix typos
- :pmmbug:`2515`: ``pmm-admin annotate`` - more than 1 annotation

How to get PMM
================================================================================

PMM is available for installation using three methods:

- On Docker Hub – ``docker pull percona/pmm-server`` https://www.percona.com/doc/percona-monitoring-and-management/deploy/server/docker.html
- AWS Marketplace – https://www.percona.com/doc/percona-monitoring-and-management/deploy/server/ami.html
- Open Virtualization Format (OVF) – https://www.percona.com/doc/percona-monitoring-and-management/deploy/server/virtual-appliance.html

Help us improve our software quality by reporting any bugs you encounter using our `bug tracking system`_.

.. _`release announcement`: https://www.percona.com/blog/2018/05/23/percona-monitoring-and-management-1-11-0-is-now-available/
.. _`bug tracking system`: https://jira.percona.com/secure/Dashboard.jspa

.. |release| replace:: 1.11.0

.. include:: .res/replace/name.txt

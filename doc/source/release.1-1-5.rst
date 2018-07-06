.. _pmm/release/1-1-5:

|pmm.name| |release|
********************************************************************************

:Date: June 21, 2017

For install and upgrade instructions, see :ref:`deploy-pmm`.

Changes in |pmm-server|
================================================================================

* :pmmbug:`667`: Fixed the *Latency* graph
  in the *ProxySQL Overview* dashboard
  to plot microsecond values instead of milliseconds.

* :pmmbug:`800`: Fixed the *InnoDB Page Splits* graph
  in the *MySQL InnoDB Metrics Advanced* dashboard
  to show correct page merge success ratio.

* :pmmbug:`1007`: Added links to Query Analytics
  from *MySQL Overview* and *MongoDB Overview* dashboards.
  The links also pass selected host and time period values.

  .. note:: These links currently open QAN2,
     which is still considered experimental.

Changes in |pmm-client|
================================================================================

* :pmmbug:`931`: Fixed ``pmm-admin`` script
  when adding MongoDB metrics monitoring for secondary in replica set.

.. |release| replace:: 1.1.5

.. include:: .res/replace.txt

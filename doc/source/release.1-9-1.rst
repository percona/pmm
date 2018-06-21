:orphan: true

.. _pmm/release/1-9-1:

|pmm.name| |release|
********************************************************************************

:Date: April 12, 2018

For more information about this release, see the `release announcement`_.

This release effectively solves the problem in |qan| when the
|gui.count| column actually displayed the number of queries per
minute, not per second, as the user would expect.

|h.issues|
================================================================================

|tip.bug-fix-release| |pmm.name| |prev-release|.

.. rubric:: |h.bug-fixes|

- :pmmbug:`2364`: QPS are wrong in QAN

.. seealso::

   All releases
      :ref:`pmm/release/list`

   Latest release
      :ref:`pmm/release/latest`

   To release |prev-release|
      :ref:`pmm/release/1-9-0`

.. |release| replace:: 1.9.1
.. |prev-release| replace:: 1.9.0
		       
.. _`release announcement`: https://www.percona.com/blog/2018/04/12/percona-monitoring-and-management-1-9-1-is-now-available/

.. include:: .res/replace.txt


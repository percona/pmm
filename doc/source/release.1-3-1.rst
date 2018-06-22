.. _pmm/release/1-3-1:

|pmm.name| |release|
********************************************************************************

:Date: September 29, 2017

|pmm.name| |release| only contains bug fixes related to usability.

For install and upgrade instructions, see :ref:`deploy-pmm`.

.. rubric:: Bug fixes

* :pmmbug:`1271`: In |qan|, when the user selected a database host with no
  queries, the query monitor could still show metrics.
* :pmmbug:`1512`: When reached from |grafana|, |qan|
  would open its home page. Now, |qan| opens and automatically
  selects the database host and time range active in |grafana|.
* :pmmbug:`1523`: User defined |prometheus| memory settings were not
  honored, potentially causing performance issues in high load
  environments.

Other bug fixes in this release: :pmmbug:`1452`, :pmmbug:`1515`.

.. |release| replace:: 1.3.1

.. include:: .res/replace/name.txt

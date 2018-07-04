.. _pmm/release/1-5-2:

|pmm.name| |release|
********************************************************************************

:Date: December 5, 2017

|percona| announces the release of |pmm.name| |release|. This release contains
fixes to bugs found after |pmm.name| |prev-release| was released.

.. rubric:: Bug fixes

- :pmmbug:`1790`: |qan| would display query metrics even for a host that was not
  configured for |opt.mysql-queries| or |opt.mongodb-queries|. We have fixed the
  behaviour to display an appropriate warning message when there are no query
  metrics for the selected host.
- :pmmbug:`1826`: If |pmm-server| 1.5.0 is deployed via |docker|, the
  |gui.update| button would not upgrade the instance to a later release.
- :pmmbug:`1830`: If |pmm-server| 1.5.0 is deployed via |abbr.ami| instance,
  the |gui.update| button would not upgrade the instance to a later release.

.. |release|      replace:: 1.5.2
.. |prev-release| replace:: 1.5.1

.. include:: .res/replace.txt


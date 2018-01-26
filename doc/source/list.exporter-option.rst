:orphan: true

.. _pmm/list.exporter-option:

================================================================================
Exporter Options
================================================================================

This is a list of options that you may pass directly to the |prometheus| exporter
when running |pmm-admin.add|.

The exporter options are passed along with the monitoring service
after two dashes (:code:`--`).

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.add.mongodb-metrics.mongodb-tls+
   :end-before: #+end-block

.. toctree::
   :glob:
   :maxdepth: 1

   exporter-option.*

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
   

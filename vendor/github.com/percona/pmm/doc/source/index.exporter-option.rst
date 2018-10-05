:orphan: true

.. _pmm.list.exporter:

Exporters Overview
********************************************************************************

This is a list of exporters that |pmm.name| uses to provides metrics from the
supported systems. For each exporter, you may find informatioih about the
options that can be passed directly to the |prometheus|.  when running
|pmm-admin.add|.

The exporter options are passed along with the monitoring service after two
dashes (:code:`--`).

.. include:: .res/code/pmm-admin.add.mongodb-metrics.mongodb-tls.txt

.. toctree::
   :glob:
   :maxdepth: 1

   section.exporter.*

.. include:: .res/replace.txt


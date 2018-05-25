.. _dashboard.prometheus-exporters-overview:

|prometheus| Exporters Overview
================================================================================

The |prometheus| Exporters Overview dashboard provides the summary of
how exporters are used across the selected hosts.

.. seealso::

   |percona| Database Performance Blog

      `Understand Your Prometheus Exporters with Percona Monitoring and Management (PMM)
      <https://www.percona.com/blog/2018/02/20/understand-prometheus-exporters-percona-monitoring-management-pmm/>`_
   
   |prometheus| Documentation

      `Exporters and integrations <https://prometheus.io/docs/instrumenting/exporters/>`_

.. Metrics

.. contents::
   :local:

.. _dashboard.prometheus-exporters-overview.summary:

|prometheus| Exporters Summary
--------------------------------------------------------------------------------

This section provides a summary of how exporters are used across the selected
hosts. It includes the average usage of CPU and memory as well as the number of
hosts being monitored and the total number of running exporters.

.. rubric:: Metrics in this section

- Avg CPU Usage per Host
- Avg Memory Usage per Host
- Monitored Hosts
- Exporters Running

.. note::

   The CPU usage and memory usage do not include the additional CPU and memory
   usage required to produce metrics by the application or operating system.

.. _dashboard.prometheus-exporters-overview.resource-usage-by-host:

|prometheus| Exporters Resource Usage by Host
--------------------------------------------------------------------------------

This section shows how resources, such as CPU and memory, are being used by the
exporters for the selected hosts.

.. rubric:: Metrics in this section

- CPU Usage
- Memory Usage

.. _dashboard.prometheus-exporters-overview.resource-usage-by-type:

|prometheus| Exporters Resource Usage by Type
--------------------------------------------------------------------------------

This section shows how resources, such as CPU and memory, are being used by the
exporters for host types: |mysql|, |mongodb|, |proxysql|, and the system.

.. rubric:: Metrics in this section

- CPU Cores Used
- Memory Usage

.. _dashboard.prometheus-exporters-overview.hosts:

List of Hosts
--------------------------------------------------------------------------------

At the bottom, this dashboard shows details for each running host. You can click
the value of the *CPU Used*, *Memory Used*, or *Exporters Running* column to
open the :ref:`dashboard.prometheus-exporter-status` for further analysis.

.. include:: .res/replace/name.txt








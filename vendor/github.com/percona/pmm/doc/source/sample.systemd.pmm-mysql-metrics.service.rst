:orphan: true

.. _sample.systemd:

Examples of the |systemd| Unit File
********************************************************************************

This page contains examples of setting up the |systemd| unit file.

.. _sample.systemd.unit-file.ssl-option:

:ref:`Default systemd unit file with SSL related options highlighted <sample.systemd.unit-file.ssl-option>`
===========================================================================================================

If the |systemd| unit file contains options related to SSL the
communication between the |prometheus| exporter and the monitored
system occurs via the HTTPS protocol.

.. include:: .res/code/systemd.pmm-mysql-metrics-service.highlight-ssl.txt

Remove the SSL related options to disable HTTPS for the exporter.

.. include:: .res/code/systemd.pmm-mysql-metrics-service.remove-ssl.txt

.. include:: .res/replace.txt

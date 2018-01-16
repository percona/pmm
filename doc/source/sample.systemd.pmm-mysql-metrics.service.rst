:orphan: true

.. _sample.systemd:

===============================================================================
Examples of the |systemd| Unit File
===============================================================================

This page contains examples of setting up the |systemd| unit file.

Default |systemd| unit file with SSL related options highlighted
================================================================================

If the |systemd| unit file contains options related to SSL the
communication between the |prometheus| exporter and the monitored
system occurs via the HTTPS protocol.

.. include:: .res/code/sh.org
   :start-after: +systemd.pmm-mysql-metrics-service.+highlight-ssl+
   :end-before: #+end-block

Remove the SSL related options to disable HTTPS for the exporter.

.. include:: .res/code/sh.org
   :start-after: +systemd.pmm-mysql-metrics-service.+remove-ssl+
   :end-before: #+end-block

.. include:: .res/replace/option.txt
.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt

.. _pmm.pmm-admin.monitoring-service.pass-parameter:

`Passing options to the exporter <pmm-admin.html#pmm-pmm-admin-monitoring-service-pass-parameter>`_
----------------------------------------------------------------------------------------------------

|pmm-admin.add| sends all options which follow :option:`--` (two consecutive
dashes delimited by whitespace) to the |prometheus| exporter that the given
monitoring services uses. Each exporter has its own set of options.

|tip.run-all.root|.

.. include:: .res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example.txt

.. include:: .res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example2.txt

The section :ref:`pmm.list.exporter` contains all option
grouped by exporters.

.. include:: ../.res/replace.txt

.. _pmm-admin.external.exporters:

--------------------------------------------------------------------------------
`Using External Exporters <pmm.admin.external.exporters>`_
--------------------------------------------------------------------------------

.. _pmm.pmm-admin.external-monitoring-service.adding:

`Adding external monitoring services <pmm-admin.html#pmm-pmm-admin-external-monitoring-service-adding>`_
----------------------------------------------------------------------------------------------------------

The |pmm-admin.add| command is also used to add external :ref:`monitoring
services <External-Monitoring-Service>`. This command adds an external
monitoring service assuming that the underlying |prometheus| exporter is already
set up and accessible. The default scrape timeout is 10 seconds, and the
interval equals to 1 minute.

To add an external monitoring service use the |opt.external-service| monitoring
service followed by the port number, name of a |prometheus| job. These options
are required. To specify the port number the |opt.service-port| option.

.. _pmm-admin.add.external-service.service-port.postgresql:

.. include:: ../.res/code/pmm-admin.add.external-service.service-port.postgresql.txt

By default, the |pmm-admin.add| command automatically creates the name of the
host to be displayed in the |gui.host| field of the
|dbd.advanced-data-exploration| dashboard where the metrics of the newly added
external monitoring service will be displayed. This name matches the name of the
host where |pmm-admin| is installed. You may choose another display name when
adding the |opt.external-service| monitoring service giving it explicitly after
the |prometheus| exporter name.
		
You may also use the |opt.external-metrics| monitoring service. When using this
option, you refer to the exporter by using a URL and a port number. The
following example adds an external monitoring service which monitors a
|postgresql| instance at 192.168.200.1, port 9187. After the command completes,
the |pmm-admin.list| command shows the newly added external exporter at the
bottom of the command's output:

|tip.run-this.root|

.. _pmm-admin.add.external-metrics.postgresql:

.. include:: ../.res/code/pmm-admin.add.external-metrics.postresql.txt

.. seealso::

   View all added monitoring services
      See :ref:`pmm-admin.list`

   Use the external monitoring service to add |postgresql| running on an |amazon-rds| instance
      See :ref:`use-case.external-monitoring-service.postgresql.rds`
		
.. _pmm.pmm-admin.monitoring-service.pass-parameter:

`Passing options to the exporter <pmm-admin.html#pmm-pmm-admin-monitoring-service-pass-parameter>`_
----------------------------------------------------------------------------------------------------

|pmm-admin.add| sends all options which follow :option:`--` (two consecutive
dashes delimited by whitespace) to the |prometheus| exporter that the given
monitoring services uses. Each exporter has its own set of options.

|tip.run-all.root|.

.. include:: ../.res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example.txt

.. include:: ../.res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example2.txt

The section :ref:`pmm.list.exporter` contains all option
grouped by exporters.
   

.. include:: ../.res/replace.txt

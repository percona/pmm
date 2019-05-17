.. _pmm-admin.stop:

`Stopping monitoring services with pmm-admin stop <pmm-admin.stop>`_
================================================================================

Use the |pmm-admin.stop| command to stop services
managed by this |pmm-client|.

.. _pmm-admin.stop.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.stop.service-name.options:

.. include:: ../.res/code/pmm-admin.stop.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.stop.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.stop| command:

|opt.all|
  Stop all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.stop.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to stop.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.stop.examples:

.. rubric:: EXAMPLES

* To stop all available services for this |pmm-client|:

  .. include:: ../.res/code/pmm-admin.stop.all.txt
		
* To stop all services related to |mysql|:

  .. include:: ../.res/code/pmm-admin.stop.mysql.txt
		   
* To stop only the |opt.mongodb-metrics| service:

  .. include:: ../.res/code/pmm-admin.stop.mongodb-metrics.txt
		   
For more information, run
|pmm-admin.stop|
|opt.help|.

.. include:: ../.res/replace.txt

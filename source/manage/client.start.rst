.. _pmm-admin.start:

`Starting monitoring services with pmm-admin start <pmm-admin.start>`_
================================================================================

Use the |pmm-admin.start| command to start services managed by this
|pmm-client|.

.. _pmm-admin.start.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.start.service-name.options:

.. include:: ../.res/code/pmm-admin.start.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.start.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.start| command:

|opt.all|
  Start all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.start.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to start.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.start.examples:

.. rubric:: EXAMPLES

* To start all available services for this |pmm-client|:

  .. include:: ../.res/code/pmm-admin.start.all.txt

* To start all services related to |mysql|:

  .. include:: ../.res/code/pmm-admin.start.mysql.txt
		   
* To start only the |opt.mongodb-metrics| service:

  .. include:: ../.res/code/pmm-admin.start.mongodb-metrics.txt
		
For more information, run
|pmm-admin.start|
|opt.help|.

.. include:: ../.res/replace.txt

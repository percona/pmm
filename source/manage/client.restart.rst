.. _pmm-admin.restart:

:ref:`Restarting monitoring services <pmm-admin.restart>`
=========================================================

Use the |pmm-admin.restart| command to restart services
managed by this |pmm-client|.
This is the same as running |pmm-admin.stop|_ and |pmm-admin.start|_.

.. _pmm-admin.restart.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.restart.service.name.options:

.. include:: ../.res/code/pmm-admin.restart.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.restart.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.restart| command:

|opt.all|
  Restart all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.restart.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to restart.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.restart.examples:

.. rubric:: EXAMPLES

* To restart all available services for this |pmm-client|:

  .. include:: ../.res/code/pmm-admin.restart.all.txt
		
* To restart all services related to |mysql|:

  .. include:: ../.res/code/pmm-admin.restart.mysql.txt

* To restart only the |opt.mongodb-metrics| service:

  .. include:: ../.res/code/pmm-admin.restart.mongodb-metrics.txt
		
For more information, run |pmm-admin.restart| :option:`--help`.

.. include:: ../.res/replace.txt

.. _pmm-admin.remove:
.. _pmm-admin.rm:

:ref:`Removing monitoring services <pmm-admin.remove>`
================================================================================

Use the |pmm-admin.rm| command to remove monitoring services.

.. rubric:: USAGE

|tip.run-this.root|

.. _pmm-admin.remove.options.service:

.. include:: ../.res/code/pmm-admin.rm.options.service.txt
		
When you remove a service,
collected data remains in |metrics-monitor| on |pmm-server|.
To remove the collected data, use the |pmm-admin.purge|_ command.

.. _pmm-admin.remove.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.rm| command:

|opt.all|
  Remove all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.remove.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. _pmm-admin.remove.examples:

.. rubric:: EXAMPLES

* To remove all services enabled for this |pmm-client|:

  .. include:: ../.res/code/pmm-admin.rm.all.txt
		   
* To remove all services related to |mysql|:

  .. include:: ../.res/code/pmm-admin.rm.mysql.txt

* To remove only |opt.mongodb-metrics| service:

  .. include:: ../.res/code/pmm-admin.rm.mongodb-metrics.txt
		
For more information, run |pmm-admin.rm| --help.

.. include:: ../.res/replace.txt

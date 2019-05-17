.. _pmm-admin.purge:

`Purging metrics data with pmm-admin purge <pmm-admin.purge>`_
================================================================================

Use the |pmm-admin.purge| command to purge metrics data
associated with a service on |pmm-server|.
This is usually required after you :ref:`remove a service <pmm-admin.rm>`
and do not want its metrics data to show up on graphs.

.. _pmm-admin.purge.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _pmm-admin.purge.service.name.options:

.. include:: ../.res/code/pmm-admin.purge.service.name.options.txt
		
.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.purge.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. _pmm-admin.purge.options:

.. rubric:: OPTIONS

The |pmm-admin.purge| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

For more infomation, run
|pmm-admin.purge|
|opt.help|.

.. include:: ../.res/replace.txt

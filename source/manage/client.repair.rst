.. _pmm-admin.repair:

:ref:`Removing orphaned services <pmm-admin.repair>`
================================================================================

Use the |pmm-admin.repair| command
to remove information about orphaned services from |pmm-server|.
This can happen if you removed services locally
while |pmm-server| was not available (disconnected or shut down),
for example, using the |pmm-admin.uninstall|_ command.

.. _pmm-admin.repair.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.repair.options:

.. include:: ../.res/code/pmm-admin.repair.options.txt

.. _pmm-admin.repair.options:

.. rubric:: OPTIONS

The |pmm-admin.repair| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run |pmm-admin.repair| --help.


.. include:: ../.res/replace.txt

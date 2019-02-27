.. _pmm-admin.uninstall:

:ref:`Cleaning Up Before Uninstall <pmm-admin.uninstall>`
================================================================================

Use the |pmm-admin.uninstall| command to remove all services even if
|pmm-server| is not available.  To uninstall |pmm| correctly, you first need to
remove all services, then uninstall |pmm-client|, and then stop and remove
|pmm-server|.  However, if |pmm-server| is not available (disconnected or shut
down), |pmm-admin.rm|_ will not work.  In this case, you can use
|pmm-admin.uninstall| to force the removal of monitoring services enabled for
|pmm-client|.

.. note:: Information about services will remain in |pmm-server|, and it will
   not let you add those services again.  To remove information about orphaned
   services from |pmm-server|, once it is back up and available to |pmm-client|,
   use the |pmm-admin.repair|_ command.

.. _pmm-admin.uninstall.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.uninstall.options:

.. include:: ../.res/code/pmm-admin.uninstall.options.txt

.. _pmm-admin.uninstall.options:

.. rubric:: OPTIONS

The |pmm-admin.uninstall| command does not have its own options, but you can use
:ref:`global options that apply to any other command <pmm-admin.options>`.

For more information, run
|pmm-admin.uninstall|
|opt.help|.

.. include:: ../.res/replace.txt

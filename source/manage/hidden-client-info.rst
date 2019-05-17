.. _pmm-admin.info:

`Getting information about  pmm-client with pmm-admin info <pmm-admin.info>`_
================================================================================

Use the |pmm-admin.info| command
to print basic information about |pmm-client|.

.. _pmm-admin.info.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.info.options:

.. include:: ../.res/code/pmm-admin.info.options.txt
	
.. _pmm-admin.info.options:
	
.. rubric:: OPTIONS

The |pmm-admin.info| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.info.output:

.. rubric:: OUTPUT

The output provides the following information:

* Version of |pmm-admin|
* |pmm-server| host address, and local host name and address
  (this can be configured using |pmm-admin.config|_)
* System manager that |pmm-admin| uses to manage PMM services
* Go version and runtime information

For example:

.. _code.pmm-admin.info:

.. include:: ../.res/code/pmm-admin.info.txt

For more information, run
|pmm-admin.info|
|opt.help|.

.. include:: ../.res/replace.txt

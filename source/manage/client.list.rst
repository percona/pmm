.. _pmm-admin.list:

:ref:`Listing monitoring services <pmm-admin.list>`
================================================================================

Use the |pmm-admin.list| command to list all enabled services with details.

.. _pmm-admin.list.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.list.options:

.. include:: ../.res/code/pmm-admin.list.options.txt

.. _pmm-admin.list.options:

.. rubric:: OPTIONS

The |pmm-admin.list| command supports :ref:`global options that apply to any other command
<pmm-admin.options>` and also provides a machine friendly |json| output.

|opt.json|
   list the enabled services as a |json| document. The information provided in the
   standard tabular form is captured as keys and values. The general information
   about the computer where |pmm-client| is installed is given as top level
   elements:

   .. hlist::
      :columns: 2

      * ``Version``
      * ``ServerAddress``
      * ``ServerSecurity``
      * ``ClientName``
      * ``ClientAddress``
      * ``ClientBindAddress``
      * ``Platform``

   Note that you can quickly determine if there are any errors by inspecting the
   ``Err`` top level element in the |json| output. Similarly, the ``ExternalErr`` element
   reports errors in external services.

   The ``Services`` top level element contains a list of documents which represent enabled
   monitoring services. Each attribute in a document maps to the column in the tabular
   output.

   The ``ExternalServices`` element contains a list of documents which represent
   enabled external monitoring services. Each attribute in a document maps to
   the column in the tabular output.

.. _pmm-admin.list.output:

.. rubric:: OUTPUT

The output provides the following information:

* Version of |pmm-admin|
* |pmm-server| host address, and local host name and address (this can be
  configured using |pmm-admin.config|_)
* System manager that |pmm-admin| uses to manage |pmm| services
* A table that lists all services currently managed by ``pmm-admin``, with basic
  information about each service

For example, if you enable general OS and |mongodb| metrics monitoring, output
should be similar to the following:

|tip.run-this.root|

.. _code.pmm-admin.list:

.. include:: ../.res/code/pmm-admin.list.txt


.. include:: ../.res/replace.txt

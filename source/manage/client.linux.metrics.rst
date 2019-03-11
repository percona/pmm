--------------------------------------------------------------------------------
Adding Linux metrics
--------------------------------------------------------------------------------

.. _pmm-admin-add-linux-metrics:

`Adding general system metrics service <pmm-admin-add-linux-metrics>`_
================================================================================

Use the |opt.linux-metrics| alias to enable general system metrics monitoring.

.. _pmm-admin-add-linux-metrics.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add.linux-metrics:

.. include:: ../.res/code/pmm-admin.add.linux-metrics.txt

This creates the ``pmm-linux-metrics-42000`` service
that collects local system metrics for this particular OS instance.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.add.linux-metrics.options:

.. rubric:: OPTIONS

The following option can be used with the ``linux:metrics`` alias:

|opt.force|
  Force to add another general system metrics service with a different name
  for testing purposes.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general
<pmm-admin.add-options>`.

For more information, run
|pmm-admin.add|
|opt.linux-metrics|
|opt.help|.

.. seealso::

   Default ports
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`


.. include:: ../.res/replace.txt

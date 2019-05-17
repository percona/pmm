--------------------------------------------------------------------------------
Adding a ProxySQL host
--------------------------------------------------------------------------------

.. _pmm-admin.add-proxysql-metrics:

`Adding ProxySQL metrics service <pmm-admin.add-proxysql-metrics>`_
================================================================================

Use the |opt.proxysql-metrics| alias
to enable |proxysql| performance metrics monitoring.

.. _pmm-admin.add-proxysql-metrics.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add-proxysql-metrics:

.. include:: ../.res/code/pmm-admin.add.proxysql-metrics.txt

This creates the ``pmm-proxysql-metrics-42004`` service
that collects local |proxysql| performance metrics.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.add-proxysql-metrics.options:

.. rubric:: OPTIONS

The following option can be used with the |opt.proxysql-metrics| alias:

|opt.dsn|
  Specify the ProxySQL connection DSN.
  By default, it is ``stats:stats@tcp(localhost:6032)/``.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general
<pmm-admin.add-options>`.

For more information, run
|pmm-admin.add|
|opt.proxysql-metrics|
|opt.help|.

.. seealso::

   Default ports
      :ref:`Ports <Ports>` in :ref:`pmm.glossary-terminology-reference`


.. include:: ../.res/replace.txt

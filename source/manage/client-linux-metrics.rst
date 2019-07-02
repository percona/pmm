--------------------------------------------------------------------------------
Adding Linux metrics
--------------------------------------------------------------------------------

.. _pmm-admin-add-linux-metrics:

`Adding general system metrics service <client-linux-metrics.html#pmm-admin-add-linux-metrics>`_
================================================================================================

PMM2 collects Linux metrics automatically starting from the moment when you
have `configured your node for monitoring with pmm-admin config <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/client-config.html#deploy-pmm-client-server-connecting>`_.

.. only:: showhidden

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
	      :ref:`Ports <Ports>` in :ref:`pmm.glossary-terminology-reference`

.. include:: ../.res/replace.txt

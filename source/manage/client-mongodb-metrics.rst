.. _pmm-admin.add.mongodb-metrics:

`Adding MongoDB Service Monitoring <pmm-admin.add.mongodb-metrics>`_
================================================================================

You can add MongoDB services (Metrics and Query Analytics) with the following command:

.. code-block:: text

   $ pmm-admin add mongodb --use-profiler --use-exporter  --username=pmm  --password=pmm

where username and password are credentials for the monitored MongoDB access, which will be used locally on the database host.

You can then check your MySQL and MongoDB dashboards and Query Analytics in order to view your server’s performance information.

.. only:: showhidden

	Use the |opt.mongodb-metrics| alias to enable MongoDB metrics monitoring.

	.. _pmm-admin.add.mongodb-metrics.usage:

	.. rubric:: USAGE

	.. _code.pmm-admin.add.mongodb-metrics:

	.. include:: ../.res/code/pmm-admin.add.mongodb-metrics.txt

	This creates the ``pmm-mongodb-metrics-42003`` service
	that collects local |mongodb| metrics for this particular |mongodb| instance.

	.. note:: It should be able to detect the local |pmm-client| name,
	   but you can also specify it explicitly as an argument.

	.. _pmm-admin.add.mongodb-metrics.options:

	.. rubric:: OPTIONS

	The following options can be used with the |opt.mongodb-metrics| alias:

	|opt.cluster|
	  Specify the MongoDB cluster name. 

	|opt.uri|
	  Specify the MongoDB instance URI with the following format::

	   [mongodb://][user:pass@]host[:port][/database][?options]

	  By default, it is ``localhost:27017``.

	You can also use
	:ref:`global options that apply to any other command
	<pmm-admin.options>`,
	as well as
	:ref:`options that apply to adding services in general
	<pmm-admin.add-options>`.

	For more information, run
	|pmm-admin.add|
	|opt.mongodb-metrics|
	|opt.help|.

	.. _pmm-admin.add.mongodb-metrics.cluster.monitoring:

	.. rubric:: Monitoring a cluster

	When using |pmm| to monitor a cluster, you should enable monitoring for each
	instance by using the |pmm-admin.add| command. This includes each member of
	replica sets in shards, mongos, and all configuration servers. Make sure that
	for each instance you supply the cluster name via the |opt.cluster| option and
	provide its URI via the |opt.uri| option.

	|tip.run-this.root|. This examples uses *127.0.0.1* as a URL.

	.. code-block:: bash

	   $ pmm-admin add mongodb:metrics \
	   --uri mongodb://127.0.0.1:<port>/admin <instance name> \
	   --cluster <cluster name>

	.. seealso::

	   Default ports
	      :ref:`Ports <Ports>` in :ref:`pmm.glossary-terminology-reference`
	   Essential |mongodb| configuration 
	      :ref:`pmm.qan-mongodb.conf`
	   

.. include:: ../.res/replace.txt

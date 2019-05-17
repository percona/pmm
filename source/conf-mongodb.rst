.. _pmm.qan-mongodb.conf:

Configuring |mongodb| for Monitoring in |qan.name|
********************************************************************************

In |abbr.qan|, you can monitor |mongodb| metrics and |mongodb| queries with the
|opt.mongodb-metrics| or |opt.mongodb-queries| monitoring services
accordingly. Run the |pmm-admin.add| command to use these monitoring services
(for more information, see :ref:`pmm-admin.add`).

.. _pmm.conf.mongodb.supported-version:

.. rubric:: Supported versions of |mongodb|

|qan| supports |mongodb| version 3.2 or higher.

.. contents::
   :local:
   :depth: 1

.. _pmm.qan-mongodb.conf.essential-permission.setting-up:

Setting Up the Essential Permissions
================================================================================

For |opt.mongodb-metrics| and |opt.mongodb-queries| monitoring services to be
able work in |qan|, you need to set up the |mongodb-exporter| user. This user
should be assigned the |cluster-monitor| role for the |db.admin| database and
the *read* role for the |db.local| database.

The following example that you can run in the |mongodb| shell, adds the
|mongodb-exporter| user and assigns the appropriate roles.

.. _code.pmm.qan-mongodb.conf.essential-permission.setting-up.db.get-sibling-db.create-user:

.. include:: .res/code/db.get-sibling-db.create-user.txt

Then, you need to pass the user name and password in the value of the
|opt.uri| option when adding the |opt.mongodb-metrics| monitoring
service in the |pmm-admin.add| command:

|tip.run-this.root|.

.. _pmm.qan-mongodb.conf.essential-permission.setting-up.pmm-admin.add.mongodb-metrics.uri:

.. include:: .res/code/pmm-admin.add.mongodb-metrics.uri.txt

.. seealso::

   Adding a |opt.mongodb-metrics| monitoring service
      :ref:`pmm-admin.add.mongodb-metrics`

.. _pmm.qan-mongodb.configuring.profiling.enabling:

`Enabling Profiling <conf-mongodb.html#pmm-qan-mongodb-configuring-profiling-enabling>`_
========================================================================================

For `MongoDB`_ to work correctly with |abbr.qan|, you need to enable profiling
in your |mongod| configuration. When started without profiling enabled, |qan|
displays the following warning:

.. note:: **A warning message is displayed when profiling is not enabled**

   It is required that profiling of the monitored |mongodb| databases be enabled.

   Note that profiling is not enabled by default because it may reduce the
   performance of your |mongodb| server.

.. _pmm.qan-mongodb.conf.profiling.command_line.enable:

`Enabling Profiling on Command Line <conf-mongodb.html#pmm-qan-mongodb-conf-profiling-command-line-enable>`_
-------------------------------------------------------------------------------------------------------------

You can enable profiling from command line when you start the :program:`mongod`
server. This command is useful if you start :program:`mongod` manually.

|tip.run-this.root|

.. _pmm.qan-mongodb.conf.profiling.command_line.enable.mongod.dbpath.profile.slowms.ratelimit:

.. include:: .res/code/mongod.dbpath.profile.slowms.ratelimit.txt

Note that you need to specify a path to an existing directory that stores
database files with the |opt.dbpath|. When the |opt.profile| option is set to
**2**, |mongod| collects the profiling data for all operations. To decrease the
load, you may consider setting this option to **1** so that the profiling data
are only collected for slow operations.

The |opt.slowms| option sets the minimum time for a slow operation. In the given
example, any operation which takes longer than **200** milliseconds is a slow
operation.

The |opt.rate-limit| option, which is available if you use |psmdb| instead
of |mongodb|, refers to the number of queries that the |mongodb| profiler
collects. The lower the rate limit, the less impact on the performance. However,
the accuracy of the collected information decreases as well.

.. seealso::

   |opt.rate-limit| in |psmdb| documentation
       https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html

.. _pmm.qan-mongodb.configuring.configuration-file.profiling.enabling:

`Enabling Profiling in the Configuration File <conf-mongodb.html#pmm-qan-mongodb-configuring-configuration-file-profiling-enabling>`_
-------------------------------------------------------------------------------------------------------------------------------------

If you run ``mongod`` as a service, you need to use the configuration file which
by default is |etc.mongod.conf|.

In this file, you need to locate the *operationProfiling:* section and add the
following settings:

.. _pmm.qan-mongodb.configuring.configuration-file.profiling.enabling.operationprofiling:

.. code-block:: yaml

   operationProfiling:
      slowOpThresholdMs: 200
      mode: slowOp
      rateLimit: 100

These settings affect ``mongod`` in the same way as the command line
options described in section
:ref:`pmm.qan-mongodb.conf.profiling.command_line.enable`. Note that the
configuration file is in the `YAML`_ format. In this format the indentation of
your lines is important as it defines levels of nesting.

Restart the *mongod* service to enable the settings.

.. _pmm.qan-mongodb.configuring.configuration-file.profiling.enabling.service.mongod.restart:

|tip.run-this.root|

.. include:: .res/code/service.mongod.restart.txt

.. admonition:: |related-information| 

   |mongodb| Documentation: Enabling Profiling
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/
   |mongodb| Documentation: Profiling Mode
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.mode
   |mongodb| Documentation: SlowOpThresholdMd option
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.slowOpThresholdMs
   |mongodb| Documentation: Profiler Overhead (from |mongodb| documentation)
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/#profiler-overhead
   Documentation for Percona Server for MongoDB: Profiling Rate Limit
      https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html

.. _MongoDB: https://www.mongodb.com
.. _YAML: http://yaml.org/spec/

.. include:: .res/replace.txt

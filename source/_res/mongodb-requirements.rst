Configuring MongoDB for Monitoring in PMM Query Analytics
=========================================================

In Query Analytics, you can monitor MongoDB metrics and queries. Run the
``pmm-admin add`` command to use these monitoring services
(for more information, see :ref:`Adding MongoDB Service Monitoring <pmm.pmm-admin.mongodb.add-mongodb>`).

.. rubric:: Supported versions of MongoDB

Query Analytics supports MongoDB version 3.2 or higher.

Setting Up the Required Permissions
===================================

For MongoDB monitoring services to be able work in Query Analytics, you need to
set up the ``mongodb_exporter`` user. This user should be assigned the
*clusterMonitor* and *readAnyDatabase* roles for the ``admin`` database.

The following is an example you can run in the MongoDB shell, to add the
``mongodb_exporter`` user and assign the appropriate roles:

.. include:: /.res/code/db.get-sibling-db.create-user.txt

Enabling Profiling
==================

For `MongoDB`_ to work correctly with Query Analytics, you need to enable profiling
in your ``mongod`` configuration. When started without profiling enabled, Query Analytics
displays the following warning:

.. note:: **A warning message is displayed when profiling is not enabled**

   It is required that profiling of the monitored MongoDB databases be enabled, however
   profiling is not enabled by default because it may reduce the performance of your
   MongoDB server.


Enabling Profiling on Command Line
----------------------------------

You can enable profiling from command line when you start the ``mongod``
server. This command is useful if you start ``mongod`` manually.

Run this command as root or by using the ``sudo`` command


.. include:: /.res/code/mongod.dbpath.profile.slowms.ratelimit.txt

Note that you need to specify a path to an existing directory that stores
database files with the ``--dpbath``. When the ``--profile`` option is set to
**2**, ``mongod`` collects the profiling data for all operations. To decrease the
load, you may consider setting this option to **1** so that the profiling data
are only collected for slow operations.

The ``--slowms`` option sets the minimum time for a slow operation. In the
given example, any operation which takes longer than **200** milliseconds is a
slow operation.

The ``--rateLimit`` option, which is available if you use PSMDB instead
of MongoDB, refers to the number of queries that the MongoDB profiler
collects. The lower the rate limit, the less impact on the performance.
However, the accuracy of the collected information decreases as well.

.. seealso::

   ``--rateLimit`` in `PSMDB documentation
   <https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html>`_


Enabling Profiling in the Configuration File
--------------------------------------------

If you run ``mongod`` as a service, you need to use the configuration file
which by default is ``/etc/mongod.conf``.

In this file, you need to locate the *operationProfiling:* section and add the
following settings:

.. code-block:: yaml

   operationProfiling:
      slowOpThresholdMs: 200
      mode: slowOp
      rateLimit: 100

These settings affect ``mongod`` in the same way as the command line
options. Note that the
configuration file is in the `YAML`_ format. In this format the indentation of
your lines is important as it defines levels of nesting.

Restart the *mongod* service to enable the settings.

Run this command as root or by using the ``sudo`` command

.. include:: /.res/code/service.mongod.restart.txt

.. admonition:: Related Information

   MongoDB Documentation: Enabling Profiling
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/
   MongoDB Documentation: Profiling Mode
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.mode
   MongoDB Documentation: SlowOpThresholdMd option
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.slowOpThresholdMs
   MongoDB Documentation: Profiler Overhead (from MongoDB documentation)
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/#profiler-overhead
   Documentation for Percona Server for MongoDB: Profiling Rate Limit
      https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html

.. _MongoDB: https://www.mongodb.com
.. _YAML: http://yaml.org/spec/

.. _pmm/qan/mongodb/conf:

================================================================================
Configuring profiling in MongoDB
================================================================================

For `MongoDB`_ to work correctly with |qan.intro|, you need to enable
profiling in your |mongod| configuration. When started without
profiling enabled, |qan| displays the following warning:

.. note:: **The warning message displayed when profiling is not enabled**

   It is required that profiling of monitored |mongodb| databases be enabled.

   Note that profiling is not enabled by default because it may reduce the
   performance of your |mongodb| server.

.. _pmm/qan/mongodb/conf/profiling.command_line.enable:

Enabling Profiling on Command Line
--------------------------------------------------------------------------------

You can enable profiling from command line when you start the :program:`mongod`
server. This command is useful if you start :program:`mongod` manually.

|tip.run-this.root|

.. include:: .res/code/sh.org
   :start-after: +mongod.dbpath.profile.slowms.ratelimit+
   :end-before: #+end-block

Note that you need to specify a path to an existing directory that stores
database files with the |opt.dbpath|. When the |opt.profile| option
is set to **1**, |mongod| only collects the profiling data for slow
operations. The |opt.slowms| option sets the minimum time for a slow
operation. In the given example, any operation which takes longer than **200**
milliseconds is a slow operation.

The |opt.rate-limit| option refers to the number of queries that the |mongodb|
profiler collects. The lower the rate limit, the less impact on the
performance. However, the accuracy of the collected information decreases as
well.

Enabling Profiling in the Configuration File
--------------------------------------------------------------------------------

If you run |mongod| as a service, you need to use the configuration file which
by default is |etc.mongod.conf|.

In this file, you need to locate the *operationProfiling:* section and add the
following settings:

.. include:: .res/code/yaml.org
   :start-after: +operationprofiling+
   :end-before: #+end-block


These settings affect :program:`mongod` in the same way as the command line
options described in section
:ref:`pmm/qan/mongodb/conf/profiling.command_line.enable`. Note that the
configuration file is in the `YAML`_ format. In this format the indentation of
your lines is important as it defines levels of nesting.

Restart the *mongod* service to enable the settings.

.. code-block:: bash

   $ sudo service mongod restart

.. seealso:: 

   .. rubric:: *Official MongoDB documentation:*
   
   Enabling Profiling
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/
   Profiling Mode
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.mode
   The :option:`SlowOpThresholdMd` option
      https://docs.mongodb.com/manual/reference/configuration-options/#operationProfiling.slowOpThresholdMs
   Profiler Overhead
      https://docs.mongodb.com/manual/tutorial/manage-the-database-profiler/#profiler-overhead
      
   .. rubric:: *Percona documentation:*

   Profiling Rate Limit
      https://www.percona.com/doc/percona-server-for-mongodb/LATEST/rate-limit.html


.. _YAML: http://yaml.org/spec/
.. _MongoDB: https://www.mongodb.com/

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/option.txt
.. include:: .res/replace/fragment.txt

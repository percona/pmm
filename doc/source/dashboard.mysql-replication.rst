.. _dashboard.mysql-replication:

|mysql| Replication
================================================================================

.. contents::
   :local:

.. _dashboard.mysql-replication.io-thread-running:

IO Thread Running
--------------------------------------------------------------------------------

This metric shows if the IO Thread is runnig or not. It only applies to a slave
host.

.. include:: .res/contents/io-thread.what-is.txt

.. rubric:: Possible values

Yes

   The thread is running and is connected to a replication master

No

   The thread is not running because it is not lauched yet or because an error
   has occured connecting to the master host

Connecting

   The thread is running but is not connected to a replication master

No value

   The host is not configured to be a replication slave

IO Thread Running is one of the parameters that the command
|sql.show-slave-status| returns.

.. seealso::

   |mysql| Documentation

      - `Replication <https://dev.mysql.com/doc/refman/5.7/en/replication.html>`_
      - `SHOW SLAVE STATUS Syntax <https://dev.mysql.com/doc/refman/5.7/en/show-slave-status.html>`_
      - `IO Thread states <https://dev.mysql.com/doc/refman/5.7/en/slave-io-thread-states.html>`_
 
|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.sql-thread-running:

SQL Thread Running
--------------------------------------------------------------------------------

This metric shows if the SQL thread is running or not. It only applies to a
slave host.

.. include:: .res/contents/io-thread.what-is.txt
   
.. rubric:: Possibile values

Yes

   SQL Thread is running and is applying events from the realy log to the local
   slave host

No

   SQL Thread is not running because it is not launched yet or because of an
   errror occurred while applying an event to the local slave host

.. seealso::

   |mysql| Documentation:

      - `Replication <https://dev.mysql.com/doc/refman/5.7/en/replication.html>`_
      - `SHOW SLAVE STATUS Syntax <https://dev.mysql.com/doc/refman/5.7/en/show-slave-status.html>`_
      - `SQL Thread states <https://dev.mysql.com/doc/refman/5.7/en/slave-sql-thread-states.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.replication-error-no:

Replication Error No
--------------------------------------------------------------------------------

This metric shows the number of the last error in the SQL Thread encountered
that caused the stopping of the replication process.

One of the most frequent errors that can stop the replication is *Error: 1022
Duplicate Key Entry*. In occurs if someone has previously inserted a record on
the slave (erroneously) that is generating now a conflict on the primary key
that is coming from the master. SQL Thread catches the error and stops
replication in order to avoid data corruption.

.. seealso::

   |mysql| Documentation:

      `A complete list of error codes <https://dev.mysql.com/doc/refman/5.7/en/error-messages-server.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.read-only:

Read only
--------------------------------------------------------------------------------

This metric indicates if the host is configured to be a *read only* system or not.

.. rubric:: Possible values

Yes

   The slave host permits no client updates except from users who have the SUPER
   privilege or the REPLICATION SLAVE privilege.

   This kind of configuration is tipically used for slave hosts in a replication
   environment to avoid a user can inadvertently or voluntarily modify data
   causing inconsistencies and stopping the replication process.

No

   The slave host is not configured in *read only* mode.

.. seealso::

   |mysql| Documentation:

      `Replication <https://dev.mysql.com/doc/refman/5.7/en/replication.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.mysql-replication-delay:

MySQL Replication Delay
--------------------------------------------------------------------------------

This metric shows the number of seconds the slave host is late compared to the
master host. It only applies to a slave host.

Since the replication process applies the data modifications on the slave
asyncronously, it could happen that the slave replicates events after some
time. The main reasons are:

- network latency
- since replication can usually rely on a single thread, or a limited number of
  threads, to apply data modifications, the slave host can’t have an elevated
  grade of concurrency. If a query needs a lot of time to be applied because it
  involves a huge number of records, all the following queries queued on the
  same thread must wait for its completion prior to proceed.

Generally it is not a big issue if sometimes the replication process lags a
little, but it needs to be taken care of if the delay increases constantly.

In case the latency is very high or increases constantly the slave host must be
boosted or it needs to use the multi-threaded replication.

Sharding your database or switching to a different replication topology could be
a valid options in case it is impossible to reduce the latency even after the
adoption of a powerful machine and the use of multi-threaded replication.

The optimal value is 0 (zero). In this case we cannot consider the
process to be perfectly *in sync* either - it simply means that the latency is
negligible.

.. seealso::

   Related metrics:

      - :ref:`dashboard.mysql-replication.relay-log-space`

   |mysql| Documentation

      - `SHOW SLAVE STATUS Syntax <https://dev.mysql.com/doc/refman/5.7/en/show-slave-status.html>`_
      - `Improving replication performance
	<https://dev.mysql.com/doc/refman/5.7/en/replication-solutions-performance.html>`_
      - `Replication Slave Options and Variables
	<https://dev.mysql.com/doc/refman/5.7/en/replication-options-slave.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.binlog-size:

Binlog Size
--------------------------------------------------------------------------------

This metric shows the overall dimension of the binary log files. The binary log
(also known as the binlog) contains *events* that describe database changes:
table creations, table alterations such as index creations, updates, inserts,
deletes and other useful information to let the replicaton process work
properly.

The binlog is the file that is read by the slave hosts in order to replicate
locally any modification on the data and on the table structures.

.. include:: .res/contents/binlog-file.info.txt
	     
.. seealso::

   |mysql| Documentation:

      - `The binary log <https://dev.mysql.com/doc/refman/5.7/en/binary-log.html>`_
      - `Configuring replication <https://dev.mysql.com/doc/refman/5.7/en/replication-configuration.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.binlog-data-written-hourly:
 
Binlog Data Written Hourly
--------------------------------------------------------------------------------

This metric shows the amount of data written hourly to the binlog files during
the last 24 hours. This metric can give you an idea of how big is your
application in terms of data writes (creation, modification, deletion).

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.binlog-count:

Binlog Count
--------------------------------------------------------------------------------

This metric shows the number of binlog files on the system.

.. include:: .res/contents/binlog-file.info.txt

.. seealso::

   |mysql| Documentation:

      - `The binary log <https://dev.mysql.com/doc/refman/5.7/en/binary-log.html>`_
      - `Configuring replication <https://dev.mysql.com/doc/refman/5.7/en/replication-configuration.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.binlogs-created-hourly:

Binlogs Created Hourly
--------------------------------------------------------------------------------

This metric shows the number of binlog files created hourly during the last 24 hours.

.. include:: .res/contents/binlog-file.info.txt

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.relay-log-space:

Relay Log Space
--------------------------------------------------------------------------------

This metric shows the overall dimension of the relay log files. It only applies
to a slave host.

The relay log consists of a set of numbered files containing the events to be
executed on the slave host in order to replicate database changes.

The relay log has the same format as the binlog.

There can be multiple relay log files depending on the rotation policy adopted
(using the configuration variable |opt.max-relay-log-size|).

As soon as the SQL thread completes to execute all events in the relay log file,
the file is deleted.

If this metric contains a high value, the variable |opt.max-relay-log-file| is
high too. Generally, this not a serious issue. If the value of this metric is
constantly increased, the slave is delaying too much in applying the events.

Treat this metric in the same way as the
:ref:`dashboard.mysql-replication.mysql-replication-delay` metric.

.. seealso::

   |mysql| Documentation:

      - `The Slave Relay Log <https://dev.mysql.com/doc/refman/5.7/en/slave-logs-relaylog.html>`_

|view-all-metrics| |this-dashboard|

.. _dashboard.mysql-replication.relay-log-written-hourly:

Relay Log Written Hourly
--------------------------------------------------------------------------------

This metric shows the amount of data written hourly into relay log files during
the last 24 hours.

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard.mysql-replication`

.. include:: .res/replace/option.txt
.. include:: .res/replace/fragment.txt
.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt

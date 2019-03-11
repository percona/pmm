.. _pmm-admin.add-mysql-queries:

`Understanding MySQL query analytics service <pmm-admin.add-mysql-queries>`_
================================================================================

Use the |opt.mysql-queries| alias to enable |mysql| query analytics.

.. _pmm-admin.add-mysql-queries.usage:

.. rubric:: USAGE

.. include:: ../.res/code/pmm-admin.add.mysql-queries.txt
		 
This creates the ``pmm-mysql-queries-0`` service
that is able to collect |qan| data for multiple remote |mysql| server instances.

The |pmm-admin.add| command is able to detect the local |pmm-client|
name, but you can also specify it explicitly as an argument.

.. important::

   If you connect |mysql| Server version 8.0, make sure it is started
   with the |opt.default-authentication-plugin| set to the value
   **mysql_native_password**.

   You may alter your PMM user and pass the authentication plugin as a parameter:

   .. include:: ../.res/code/alter.user.identified.with.by.txt
   
   .. seealso::

      |mysql| Documentation: Authentication Plugins
         https://dev.mysql.com/doc/refman/8.0/en/authentication-plugins.html
      |mysql| Documentation: Native Pluggable Authentication
         https://dev.mysql.com/doc/refman/8.0/en/native-pluggable-authentication.html
	 
.. _pmm-admin.add-mysql-queries.options:

.. rubric:: OPTIONS

The following options can be used with the |opt.mysql-queries| alias:

|opt.create-user|
  Create a dedicated |mysql| user for |pmm-client| (named ``pmm``).

|opt.create-user-maxconn|
  Specify maximum connections for the dedicated |mysql| user (default is 10).

|opt.create-user-password|
  Specify password for the dedicated |mysql| user.

|opt.defaults-file|
  Specify path to :file:`my.cnf`.

|opt.disable-queryexamples|
  Disable collection of query examples.

|opt.slow-log-rotation|

  Do not manage |slow-log| files by using |pmm|. Set this option to *false* if
  you intend to manage |slow-log| files by using a third party tool.  The
  default value is *true*

  .. seealso::

     Example of disabling the slow log rotation feature and using a third party tool
        :ref:`use-case.slow-log-rotation`


  .. admonition:: |related-information|

     |percona| Database Performance Blog: Rotating MySQL Slow Logs Safely
        https://www.percona.com/blog/2013/04/18/rotating-mysql-slow-logs-safely/

     |percona| Database Performance Blog: Log Rotate and the (Deleted) MySQL Log File Mystery
        https://www.percona.com/blog/2014/11/12/log-rotate-and-the-deleted-mysql-log-file-mystery/

|opt.force|
  Force to create or update the dedicated |mysql| user.

|opt.host|
  Specify the |mysql| host name.

|opt.password|
  Specify the password for |mysql| user with admin privileges.

|opt.port|
  Specify the |mysql| instance port.

|opt.query-source|
  Specify the source of data:

  * ``auto``: Select automatically (default).
  * ``slowlog``: Use the slow query log.
  * ``perfschema``: Use Performance Schema.

|opt.retain-slow-logs|
   Specify the maximum number of files of the |slow-log| to keep automatically.
   The default value is 1 file.

|opt.socket|
  Specify the |mysql| instance socket file.

|opt.user|
  Specify the name of |mysql| user with admin privileges.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

.. seealso::

   Default ports
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`

.. _pmm-admin.add-mysql-queries.detailed-description:

.. rubric:: DETAILED DESCRIPTION

When adding the |mysql| query analytics service, the |pmm-admin| tool
will attempt to automatically detect the local |mysql| instance and
|mysql| superuser credentials.  You can use options to provide this
information, if it cannot be detected automatically.

You can also specify the |opt.create-user| option to create a dedicated
``pmm`` user on the |mysql| instance that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the |mysql| superuser.

.. seealso::

   More information about |mysql| users with |pmm|
      :ref:`pmm.conf-mysql.user-account.creating`

For example, to set up remote monitoring of |qan| data on a |mysql| server
located at 192.168.200.2, use a command similar to the following:

.. _code.pmm-admin.add-mysql-queries.user.password.host.create-user:

.. include:: ../.res/code/pmm-admin.add.mysql-queries.user.password.host.create-user.txt
		
|qan| can use either the |slow-query-log| or |perf-schema| as the source.
By default, it chooses the |slow-query-log| for a local |mysql| instance
and |perf-schema| otherwise.
For more information about the differences, see :ref:`perf-schema`.

You can explicitely set the query source when adding a |qan| instance
using the |opt.query-source| option.

For more information, run
|pmm-admin.add|
|opt.mysql-queries|
|opt.help|.

.. seealso::

   How to set up |mysql| for monitoring?
      :ref:`conf-mysql`


.. include:: ../.res/replace.txt

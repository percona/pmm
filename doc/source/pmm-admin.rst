.. _pmm-admin:

Managing |pmm-client|
********************************************************************************

Use the |pmm-admin| tool to manage |pmm-client|.

|chapter.toc|

.. contents::
   :local:
   :depth: 2

.. _pmm-admin.usage:

.. rubric:: USAGE

.. code-block:: text

   pmm-admin [OPTIONS] [COMMAND]

.. note:: The |pmm-admin| tool requires root access
   (you should either be logged in as a user with root privileges
   or be able to run commands with |sudo|).

To view all available commands and options,
run |pmm-admin| without any commands or options:

.. code-block:: bash

   $ sudo pmm-admin

.. _pmm-admin.options:

.. rubric:: OPTIONS

The following options can be used with any command:

``-c``, |opt.config-file|
  Specify the location of |pmm| configuration file
  (default :file:`/usr/local/percona/pmm-client/pmm.yml`).

``-h``, |opt.help|
  Print help for any command and exit.

``-v``, |opt.version|
  Print version of |pmm-client|.

|opt.verbose|
  Print verbose output.

.. _pmm-admin.commands:

.. rubric:: COMMANDS

|pmm-admin.add|_
  Add a monitoring service.

:ref:`pmm-admin.annotate`
  Add an annotation

|pmm-admin.check-network|_
  Check network connection between |pmm-client| and |pmm-server|.

|pmm-admin.config|_
  Configure how |pmm-client| communicates with |pmm-server|.

|pmm-admin.help|_
  Print help for any command and exit.

|pmm-admin.info|_
  Print information about |pmm-client|.

|pmm-admin.list|_
  List all monitoring services added for this |pmm-client|.

|pmm-admin.ping|_
  Check if |pmm-server| is alive.

|pmm-admin.purge|_
  Purge metrics data on |pmm-server|.

|pmm-admin.remove|_, |pmm-admin.rm|_
  Remove monitoring services.

|pmm-admin.repair|_
  Remove orphaned services.

|pmm-admin.restart|_
  Restart monitoring services.

|pmm-admin.show-passwords|_
  Print passwords used by |pmm-client| (stored in the configuration file).

|pmm-admin.start|_
  Start monitoring service.

|pmm-admin.stop|_
  Stop monitoring service.

|pmm-admin.uninstall|_
  Clean up |pmm-client| before uninstalling it.

.. _pmm-admin.add:

`Adding monitoring services <pmm-admin.html#pmm-admin-add>`_
================================================================================

Use the |pmm-admin.add| command to add monitoring services.

.. _pmm-admin.add.usage:

.. rubric:: USAGE

.. code-block:: bash

   $ pmm-admin add [OPTIONS] [SERVICE]

When you add a monitoring service |pmm-admin| automatically creates
and sets up a service in the operating system. You can tweak the
|systemd| configuration file and change its behavior.
   
For example, you may need to disable the HTTPS protocol for the
|prometheus| exporter associated with the given service. To accomplish this
task, you need to remove all SSL related options.

|tip.run-all.root|:

1. Open the |systemd| unit file associated with the
   monitoring service that you need to change, such as
   |pmm-mysql-metrics.service|.

   .. include:: .res/code/cat.etc-systemd-system-pmm-mysql-metrics.txt
   
#. Remove the SSL related configuration options (key, cert) from the
   |systemd| unit file or `init.d` startup
   script. :ref:`sample.systemd` highlights the SSL related options in
   the |systemd| unit file.

   The following code demonstrates how you can remove the options
   using the |sed| command. (If you need more information about how
   |sed| works, see the documentation of your system).
   
   .. include:: .res/code/sed.e.web-ssl.pmm-mysql-metrics-service.txt
   
#. Reload |systemd|:

   .. include:: .res/code/systemctl.daemon-reload.txt

#. Restart the monitoring service by using |pmm-admin.restart|:

   .. include:: .res/code/pmm-admin.restart.mysql-metrics.txt

.. _pmm-admin.add-options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.add| command:

|opt.dev-enable|
  Enable experimental features.

|opt.disable-ssl|
  Disable (otherwise enabled) SSL for the connection between |pmm-client| and
  |pmm-server|. Turning off SSL encryption for the data acquired from some
  objects of monitoring allows to decrease the overhead for a |pmm-server|
  connected with a lot of nodes.

|opt.service-port|

  Specify the :ref:`service port <service-port>`.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

.. _pmm-admin.add.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`,
along with any relevant additional arguments.

For more information, run
|pmm-admin.add|
|opt.help|.


.. contents::
   :local:

.. _pmm.pmm-admin.external-monitoring-service.adding:

`Adding external monitoring services <pmm-admin.html#pmm-pmm-admin-external-monitoring-service-adding>`_
----------------------------------------------------------------------------------------------------------

The |pmm-admin.add| command is also used to add :ref:`external monitoring
services <External-Monitoring-Service>`. This command adds an external
monitoring service assuming that the underlying |prometheus| exporter is already
set up and accessible. The default scrape timeout is 10 seconds, and the
interval equals to 1 minute.

To add an external monitoring service use the |opt.external-service| monitoring
service followed by the port number, name of a |prometheus| job. These options
are required. To specify the port number the |opt.service-port| option.

.. _pmm-admin.add.external-service.service-port.postgresql:

.. include:: .res/code/pmm-admin.add.external-service.service-port.postgresql.txt

By default, the |pmm-admin.add| command automatically creates the name of the
host to be displayed in the |gui.host| field of the
|dbd.advanced-data-exploration| dashboard where the metrics of the newly added
external monitoring service will be displayed. This name matches the name of the
host where |pmm-admin| is installed. You may choose another display name when
adding the |opt.external-service| monitoring service giving it explicitly after
the |prometheus| exporter name.
		
You may also use the |opt.external-metrics| monitoring service. When using this
option, you refer to the exporter by using a URL and a port number. The
following example adds an external monitoring service which monitors a
|postgresql| instance at 192.168.200.1, port 9187. After the command completes,
the |pmm-admin.list| command shows the newly added external exporter at the
bottom of the command's output:

|tip.run-this.root|

.. _pmm-admin.add.external-metrics.postgresql:

.. include:: .res/code/pmm-admin.add.external-metrics.postresql.txt

.. seealso::

   View all added monitoring services
      See :ref:`pmm-admin.list`

   Use the external monitoring service to add |postgresql| running on an |amazon-rds| instance
      See :ref:`use-case.external-monitoring-service.postgresql.rds`
		
.. _pmm.pmm-admin.monitoring-service.pass-parameter:

`Passing options to the exporter <pmm-admin.html#pmm-pmm-admin-monitoring-service-pass-parameter>`_
----------------------------------------------------------------------------------------------------

|pmm-admin.add| sends all options which follow :option:`--` (two consecutive
dashes delimited by whitespace) to the |prometheus| exporter that the given
monitoring services uses. Each exporter has its own set of options.

|tip.run-all.root|.

.. include:: .res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example.txt

.. include:: .res/code/pmm.pmm-admin.monitoring-service.pass-parameter.example2.txt

The section :ref:`pmm.list.exporter` contains all option
grouped by exporters.
   
.. _pmm.pmm-admin.mongodb.pass-ssl-parameter:

`Passing SSL parameters to the mongodb monitoring service <pmm-admin.html#pmm-pmm-admin-mongodb-pass-ssl-parameter>`_
----------------------------------------------------------------------------------------------------------------------

SSL/TLS related parameters are passed to an SSL enabled |mongodb| server as
monitoring service parameters along with the |pmm-admin.add| command when adding
the |opt.mongodb-metrics| monitoring service.

|tip.run-this.root|

.. include:: .res/code/pmm-admin.add.mongodb-metrics.mongodb-tls.txt
   
.. list-table:: Supported SSL/TLS Parameters
   :widths: 25 75
   :header-rows: 1

   * - Parameter
     - Description
   * - |opt.mongodb-tls|
     - Enable a TLS connection with mongo server
   * - |opt.mongodb-tls-ca|  *string*
     - A path to a PEM file that contains the CAs that are trusted for server connections.
       *If provided*: MongoDB servers connecting to should present a certificate signed by one of these CAs.
       *If not provided*: System default CAs are used.
   * - |opt.mongodb-tls-cert| *string*
     - A path to a PEM file that contains the certificate and, optionally, the private key in the PEM format.
       This should include the whole certificate chain.
       *If provided*: The connection will be opened via TLS to the |mongodb| server.
   * - |opt.mongodb-tls-disable-hostname-validation|
     - Do hostname validation for the server connection.
   * - |opt.mongodb-tls-private-key| *string*
     - A path to a PEM file that contains the private key (if not contained in the :option:`mongodb.tls-cert` file).

.. include:: .res/contents/note.option.mongodb-queries.txt
       
.. include:: .res/code/mongod.dbpath.profile.slowms.ratelimit.txt

.. _pmm-admin-add-linux-metrics:

`Adding general system metrics service <pmm-admin.html#pmm-admin-add-linux-metrics>`_
--------------------------------------------------------------------------------------

Use the |opt.linux-metrics| alias to enable general system metrics monitoring.

.. _pmm-admin-add-linux-metrics.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add.linux-metrics:

.. include:: .res/code/pmm-admin.add.linux-metrics.txt

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

.. _pmm-admin-textfile-collector:

`Extending metrics with textfile collector <pmm-admin.html#pmm-admin-textfile-collector>`_
-------------------------------------------------------------------------------------------

.. versionadded:: 1.16.0

While |pmm| provides an excellent solution for system monitoring, sometimes you
may have the need for a metric that’s not present in the list of
``node_exporter`` metrics out of the box. There is a simple method to extend the
list of available metrics without modifying the ``node_exporter`` code. It is
based on the textfile collector.

Starting from version 1.16.0, this collector is enabled for the
``linux:metrics`` in |pmm-client| by default.

The default directory for reading text files with the metrics is
``/usr/local/percona/pmm-client/textfile-collector``, and the exporter reads
files from it with the ``.prom`` extension. By default it contains an example
file  ``example.prom`` which has commented contents and can be used as a
template.

You are responsible for running a cronjob or other regular process to generate
the metric series data and write it to this directory.

Example - collecting docker container information
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

This example will show you how to collect the number of running and stopped
docker containers on a host. It uses a ``crontab`` task, set with the following
lines in the cron configuration file (e.g. in ``/etc/crontab``)::

  */1 * * * *     root   echo -n "" > /tmp/docker_all.prom; /usr/bin/docker ps -a | sed -n '1!p'| /usr/bin/wc -l | sed -ne 's/^/node_docker_containers_total /p' >> /usr/local/percona/pmm-client/docker_all.prom;
  */1 * * * *     root   echo -n "" > /tmp/docker_running.prom; /usr/bin/docker ps | sed -n '1!p'| /usr/bin/wc -l | sed -ne 's/^/node_docker_containers_running_total /p' >>/usr/local/percona/pmm-client/docker_running.prom;

The result of the commands is placed into the ``docker_all.prom`` and
``docker_running.prom`` files and read by exporter.

The first command executed by cron is rather simple: the destination text file
is cleared by executing ``echo -n ""``, then a list of running and closed
containers is generated with ``docker ps -a``, and finally ``sed`` and ``wc``
tools are used to count the number of containers in this list and to form the
output file which looks like follows::

  node_docker_containers_total 2

The second command is similar, but it counts only running containers.

.. _pmm-admin.add-mysql-queries:

`Adding MySQL query analytics service <pmm-admin.html#pmm-admin-add-mysql-queries>`_
-------------------------------------------------------------------------------------

Use the |opt.mysql-queries| alias to enable |mysql| query analytics.

.. _pmm-admin.add-mysql-queries.usage:

.. rubric:: USAGE

.. include:: .res/code/pmm-admin.add.mysql-queries.txt
		 
This creates the ``pmm-mysql-queries-0`` service
that is able to collect |qan| data for multiple remote |mysql| server instances.

The |pmm-admin.add| command is able to detect the local |pmm-client|
name, but you can also specify it explicitly as an argument.

.. important::

   If you connect |mysql| Server version 8.0, make sure it is started
   with the |opt.default-authentication-plugin| set to the value
   **mysql_native_password**.

   You may alter your PMM user and pass the authentication plugin as a parameter:

   .. include:: .res/code/alter.user.identified.with.by.txt
   
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

.. include:: .res/code/pmm-admin.add.mysql-queries.user.password.host.create-user.txt
		
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

.. _pmm-admin.add-mysql-metrics:

`Adding MySQL metrics service <pmm-admin.html#pmm-admin-add-mysql-metrics>`_
--------------------------------------------------------------------------------

Use the |opt.mysql-metrics| alias to enable |mysql| metrics monitoring.

.. _pmm-admin.add-mysql-metrics.usage:

.. rubric:: USAGE

.. include:: .res/code/pmm-adin.add.mysql-metrics.txt

This creates the ``pmm-mysql-metrics-42002`` service
that collects |mysql| instance metrics.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.add-mysql-metrics.options:

.. rubric:: OPTIONS

The following options can be used with the |opt.mysql-metrics| alias:

|opt.create-user|
  Create a dedicated |mysql| user for |pmm-client| (named ``pmm``).

|opt.create-user-maxconn|
  Specify maximum connections for the dedicated |mysql| user (default is 10).

|opt.create-user-password|
  Specify password for the dedicated |mysql| user.

|opt.defaults-file|
  Specify the path to :file:`my.cnf`.

|opt.disable-binlogstats|
  Disable collection of binary log statistics.

|opt.disable-processlist|
  Disable collection of process state metrics.

|opt.disable-tablestats|
  Disable collection of table statistics.

|opt.disable-table-stats-limit|
  Specify the maximum number of tables
  for which collection of table statistics is enabled
  (by default, the limit is 1 000 tables).

|opt.disable-userstats|
  Disable collection of user statistics.

|opt.force|
  Force to create or update the dedicated |mysql| user.

|opt.host|
  Specify the |mysql| host name.

|opt.password|
  Specify the password for |mysql| user with admin privileges.

|opt.port|
  Specify the |mysql| instance port.

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

   More information about |qan.name|
      :ref:`pmm.qan`

.. _pmm-admin.add-mysql-metrics.detailed-description:

.. rubric:: DETAILED DESCRIPTION

When adding the |mysql| metrics monitoring service, the |pmm-admin| tool
attempts to automatically detect the local |mysql| instance and |mysql|
superuser credentials.  You can use options to provide this information, if it
cannot be detected automatically.

You can also specify the |opt.create-user| option to create a dedicated ``pmm``
user on the |mysql| host that you want to monitor.  This user will be given all
the necessary privileges for monitoring, and is recommended over using the
|mysql| superuser.

For example, to set up remote monitoring of |mysql| metrics on a server located
at 192.168.200.3, use a command similar to the following:

.. _code.pmm-admin.add-mysql-metrics.user.password.host.create-user:

.. include:: .res/code/pmm-admin.add.mysql-metrics.user.password.host.create-user.txt

For more information, run
|pmm-admin.add|
|opt.mysql-metrics|
|opt.help|.

.. seealso::

   How to set up |mysql| for monitoring?
      :ref:`conf-mysql`

.. _pmm-admin.add-mongodb-queries:

`Adding MongoDB query analytics service <pmm-admin.html#pmm-admin-add-mongodb-queries>`_
-----------------------------------------------------------------------------------------

Use the |opt.mongodb-queries| alias to enable |mongodb| query analytics.

.. _pmm-admin.add-mongodb-queries.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add-mongodb-queries:

.. include:: .res/code/pmm-admin.add.mongodb-queries.txt
		 
This creates the ``pmm-mongodb-queries-0`` service
that is able to collect |qan| data for multiple remote |mongodb| server instances.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.add-mongodb-queries.options:

.. rubric:: OPTIONS

The following options can be used with the |opt.mongodb-queries| alias:

|opt.uri|
  Specify the |mongodb| instance URI with the following format::

   [mongodb://][user:pass@]host[:port][/database][?options]

  By default, it is ``localhost:27017``. 

  .. important::

     In cases when the password contains special symbols like the *at* (@)
     symbol, the host might not not be detected correctly. Make sure that you
     insert the password with special characters replaced with their escape
     sequences. The simplest way is to use the :code:`encodeURIComponent` JavaScript function.
     
     For this, open the web console of your browser (usually found under
     *Development tools*) and evaluate the following expression, passing the
     password that you intend to use:

     .. code-block:: javascript

	> encodeURIComponent('$ecRet_pas$w@rd')
	"%24ecRet_pas%24w%40rd"

     .. admonition:: |related-information|

	MDN Web Docs: encodeURIComponent
	   https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

.. include:: .res/contents/note.option.mongodb-queries.txt

For more information, run
|pmm-admin.add|
|opt.mongodb-queries|
|opt.help|.

.. seealso::

   Default ports
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`

.. _pmm-admin.add.mongodb-metrics:

`Adding MongoDB metrics service <pmm-admin.html#pmm-admin-add-mongodb-metrics>`_
---------------------------------------------------------------------------------

Use the |opt.mongodb-metrics| alias to enable MongoDB metrics monitoring.

.. _pmm-admin.add.mongodb-metrics.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add.mongodb-metrics:

.. include:: .res/code/pmm-admin.add.mongodb-metrics.txt

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
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`
   Essential |mongodb| configuration 
      :ref:`pmm.qan.mongodb.conf`
   
.. _pmm-admin.add-proxysql-metrics:

`Adding ProxySQL metrics service <pmm-admin.html#pmm-admin-add-proxysql-metrics>`_
-----------------------------------------------------------------------------------

Use the |opt.proxysql-metrics| alias
to enable |proxysql| performance metrics monitoring.

.. _pmm-admin.add-proxysql-metrics.usage:

.. rubric:: USAGE

.. _code.pmm-admin.add-proxysql-metrics:

.. include:: .res/code/pmm-admin.add.proxysql-metrics.txt

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
      :ref:`Ports <Ports>` in :ref:`pmm.glossary.terminology-reference`

.. _pmm-admin.annotate:

`Adding annotations <pmm-admin.html#pmm-admin-annotate>`_
================================================================================

Use the |pmm-admin.annotate| command to set notifications about important
application events and display them on all dashboards. By using annotations, you
can conveniently analyze the impact of application events on your database.

.. _pmm-admin.annotate.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. include:: .res/code/pmm-admin.annotate.tags.txt

.. _pmm-admin.annotate.options:

.. rubric:: OPTIONS

The |pmm-admin.annotate| supports the following options:

|opt.tags|

   Specify one or more tags applicable to the annotation that you are
   creating. Enclose your tags in quotes and separate individual tags by a
   comma, such as "tag 1,tag 2".

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

.. _pmm-admin.check-network:

`Checking network connectivity <pmm-admin.html#pmm-admin-check-network>`_
================================================================================

Use the |pmm-admin.check-network| command to run tests
that verify connectivity between |pmm-client| and |pmm-server|.

.. _pmm-admin.check-network.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.check-network.options:

.. include:: .res/code/pmm-admin.check-network.options.txt
		
.. _pmm-admin.check-network.options:

.. rubric:: OPTIONS

The |pmm-admin.check-network| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.check-network.detailed-description:

.. rubric:: DETAILED DESCRIPTION

Connection tests are performed both ways, with results separated accordingly:

* ``Client --> Server``

  Pings |consul| API, |qan.name| API, and |prometheus| API
  to make sure they are alive and reachable.

  Performs a connection performance test to see the latency
  from |pmm-client| to |pmm-server|.

* ``Client <-- Server``

  Checks the status of |prometheus| endpoints
  and makes sure it can scrape metrics from corresponding exporters.

  Successful pings of |pmm-server| from |pmm-client|
  do not mean that Prometheus is able to scrape from exporters.
  If the output shows some endpoints in problem state,
  make sure that the corresponding service is running
  (see |pmm-admin.list|_).
  If the services that correspond to problematic endpoints are running,
  make sure that firewall settings on the |pmm-client| host
  allow incoming connections for corresponding ports.

.. _pmm-admin.check-network.output-example:

.. rubric:: OUTPUT EXAMPLE

.. _code.pmm-admin.check-network.output:

.. include:: .res/code/pmm-admin.check-network.output.txt

For more information, run
|pmm-admin.check-network|
|opt.help|.

.. _pmm-admin.diagnostics-for-support:

`Obtaining Diagnostics Data for Support <pmm-admin.html#pmm-admin-diagnostics-for-support>`_
=============================================================================================

|pmm-client| is able to generate a set of files for enhanced diagnostics, which
is designed to be shared with Percona Support to solve an issue faster. This
feature fetches logs, network, and the Percona Toolkit output. To perform data
collection by |pmm-client|, execute::

   pmm-admin summary

The output will be a tarball you can examine and/or attach to your Support
ticket in the Percona's `issue tracking system <https://jira.percona.com/projects/PMM/issues>`_. The single file will look like this::

   summary__2018_10_10_16_20_00.tar.gz

.. _pmm-admin.config:

`Configuring PMM Client <pmm-admin.html#pmm-admin-config>`_
================================================================================

Use the |pmm-admin.config| command to configure
how |pmm-client| communicates with |pmm-server|.

.. _pmm-admin.config.usage:

.. rubric:: USAGE

|tip.run-this.root|.

.. _code.pmm-admin.config.options:

.. include:: .res/code/pmm-admin.config.options.txt
		
.. _pmm-admin.config.options:

.. rubric:: OPTIONS

The following options can be used with the |pmm-admin.config| command:

|opt.bind-address|
  Specify the bind address,
  which is also the local (private) address
  mapped from client address via NAT or port forwarding
  By default, it is set to the client address.

|opt.client-address|
  Specify the client address,
  which is also the remote (public) address for this system.
  By default, it is automatically detected via request to server.

|opt.client-name|
  Specify the client name.
  By default, it is set to the host name.

|opt.force|
  Force to set the client name on initial setup
  after uninstall with unreachable server.

|opt.server|
  Specify the address of the |pmm-server| host.
  If necessary, you can also specify the port after colon, for example::

   pmm-admin config --server 192.168.100.6:8080

  By default, port 80 is used with SSL disabled,
  and port 443 when SSL is enabled.

|opt.server-insecure-ssl|
  Enable insecure SSL (self-signed certificate).

``--SERVER_PASSWORD``
  Specify the HTTP password configured on |pmm-server|.

|opt.server-ssl|
  Enable SSL encryption for connection to |pmm-server|.

``--SERVER_USER``
  Specify the HTTP user configured on |pmm-server| (default is ``pmm``).

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

For more information, run |pmm-admin.config| --help.

.. _pmm-admin.help:

`Getting help for any command <pmm-admin.html#pmm-admin-help>`_
================================================================================

Use the |pmm-admin.help| command to print help for any command.

.. _pmm-admin.help.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.help.command:

.. include:: .res/code/pmm-admin.help.command.txt

This will print help information and exit.  The actual command is not run
and options are ignored.

.. note:: You can also use the global |opt.h| or |opt.help| option after any
   command to get the same help information.

.. _pmm-admin.help.commands:

.. rubric:: COMMANDS

You can print help information for any :ref:`command <pmm-admin.commands>`
or :ref:`service alias <pmm-admin.service-aliases>`.

.. _pmm-admin.info:

`Getting information about PMM Client <pmm-admin.html#pmm-admin-info>`_
================================================================================

Use the |pmm-admin.info| command
to print basic information about |pmm-client|.

.. _pmm-admin.info.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.info.options:

.. include:: .res/code/pmm-admin.info.options.txt
	
.. _pmm-admin.info.options:
	
.. rubric:: OPTIONS

The |pmm-admin.info| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.info.output:

.. rubric:: OUTPUT

The output provides the following information:

* Version of |pmm-admin|
* |pmm-server| host address, and local host name and address
  (this can be configured using |pmm-admin.config|_)
* System manager that |pmm-admin| uses to manage PMM services
* Go version and runtime information

For example:

.. _code.pmm-admin.info:

.. include:: .res/code/pmm-admin.info.txt

For more information, run
|pmm-admin.info|
|opt.help|.

.. _pmm-admin.list:

`Listing monitoring services <pmm-admin.html#pmm-admin-list>`_
================================================================================

Use the |pmm-admin.list| command to list all enabled services with details.

.. _pmm-admin.list.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.list.options:

.. include:: .res/code/pmm-admin.list.options.txt

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

.. include:: .res/code/pmm-admin.list.txt

.. _pmm-admin.ping:

`Pinging PMM Server <pmm-admin.html#pmm-admin-ping>`_
================================================================================

Use the |pmm-admin.ping| command to verify connectivity with |pmm-server|.

.. _pmm-admin.ping.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.ping.options:

.. include:: .res/code/pmm-admin.ping.options.txt

If the ping is successful, it returns ``OK``.

.. _code.pmm-admin.ping:

.. include:: .res/code/pmm-admin.ping.txt

.. _pmm-admin.ping.options:

.. rubric:: OPTIONS

The |pmm-admin.ping| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run
|pmm-admin.ping|
|opt.help|.

.. _pmm-admin.purge:

`Purging metrics data <pmm-admin.html#pmm-admin-purge>`_
================================================================================

Use the |pmm-admin.purge| command to purge metrics data
associated with a service on |pmm-server|.
This is usually required after you :ref:`remove a service <pmm-admin.rm>`
and do not want its metrics data to show up on graphs.

.. _pmm-admin.purge.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _pmm-admin.purge.service.name.options:

.. include:: .res/code/pmm-admin.purge.service.name.options.txt
		
.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.purge.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. _pmm-admin.purge.options:

.. rubric:: OPTIONS

The |pmm-admin.purge| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

For more infomation, run
|pmm-admin.purge|
|opt.help|.

.. _pmm-admin.remove:
.. _pmm-admin.rm:

`Removing monitoring services <pmm-admin.html#pmm-admin-remove>`_
================================================================================

Use the |pmm-admin.rm| command to remove monitoring services.

.. rubric:: USAGE

|tip.run-this.root|

.. _pmm-admin.remove.options.service:

.. include:: .res/code/pmm-admin.rm.options.service.txt
		
When you remove a service,
collected data remains in |metrics-monitor| on |pmm-server|.
To remove the collected data, use the |pmm-admin.purge|_ command.

.. _pmm-admin.remove.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.rm| command:

|opt.all|
  Remove all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.remove.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. _pmm-admin.remove.examples:

.. rubric:: EXAMPLES

* To remove all services enabled for this |pmm-client|:

  .. include:: .res/code/pmm-admin.rm.all.txt
		   
* To remove all services related to |mysql|:

  .. include:: .res/code/pmm-admin.rm.mysql.txt

* To remove only |opt.mongodb-metrics| service:

  .. include:: .res/code/pmm-admin.rm.mongodb-metrics.txt
		
For more information, run |pmm-admin.rm| --help.

.. _pmm-admin.repair:

`Removing orphaned services <pmm-admin.html#pmm-admin-repair>`_
================================================================================

Use the |pmm-admin.repair| command
to remove information about orphaned services from |pmm-server|.
This can happen if you removed services locally
while |pmm-server| was not available (disconnected or shut down),
for example, using the |pmm-admin.uninstall|_ command.

.. _pmm-admin.repair.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.repair.options:

.. include:: .res/code/pmm-admin.repair.options.txt

.. _pmm-admin.repair.options:

.. rubric:: OPTIONS

The |pmm-admin.repair| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run |pmm-admin.repair| --help.

.. _pmm-admin.restart:

`Restarting monitoring services <pmm-admin.html#pmm-admin-restart>`_
=====================================================================

Use the |pmm-admin.restart| command to restart services
managed by this |pmm-client|.
This is the same as running |pmm-admin.stop|_ and |pmm-admin.start|_.

.. _pmm-admin.restart.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.restart.service.name.options:

.. include:: .res/code/pmm-admin.restart.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.restart.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.restart| command:

|opt.all|
  Restart all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.restart.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to restart.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.restart.examples:

.. rubric:: EXAMPLES

* To restart all available services for this |pmm-client|:

  .. include:: .res/code/pmm-admin.restart.all.txt
		
* To restart all services related to |mysql|:

  .. include:: .res/code/pmm-admin.restart.mysql.txt

* To restart only the |opt.mongodb-metrics| service:

  .. include:: .res/code/pmm-admin.restart.mongodb-metrics.txt
		
For more information, run |pmm-admin.restart| :option:`--help`.

.. _pmm-admin.show-passwords:

`Getting passwords used by PMM Client <pmm-admin.html#pmm-admin-show-passwords>`_
=================================================================================

Use the |pmm-admin.show-passwords| command to print credentials stored in the
configuration file (by default: :file:`/usr/local/percona/pmm-client/pmm.yml`).

.. _pmm-admin.show-passwords.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.show-passwords.options:

.. include:: .res/code/pmm-admin.show-passwords.options.txt

.. _pmm-admin.show-passwords.options:

.. rubric:: OPTIONS

The |pmm-admin.show-passwords| command does not have its own options, but you
can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.show-passwords.output:

.. rubric:: OUTPUT

This command prints HTTP authentication credentials and the password for the
``pmm`` user that is created on the |mysql| instance if you specify the
|opt.create-user| option when :ref:`adding a service <pmm-admin.add>`.

|tip.run-this.root|

.. _code.pmm-admin.show-passwords:

.. include:: .res/code/pmm-admin.show-passwords.txt

For more information, run |pmm-admin.show-passwords|  |opt.help|.

.. _pmm-admin.start:

`Starting monitoring services <pmm-admin.html#pmm-admin-start>`_
================================================================================

Use the |pmm-admin.start| command to start services managed by this
|pmm-client|.

.. _pmm-admin.start.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.start.service-name.options:

.. include:: .res/code/pmm-admin.start.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.start.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.start| command:

|opt.all|
  Start all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.start.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to start.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.start.examples:

.. rubric:: EXAMPLES

* To start all available services for this |pmm-client|:

  .. include:: .res/code/pmm-admin.start.all.txt

* To start all services related to |mysql|:

  .. include:: .res/code/pmm-admin.start.mysql.txt
		   
* To start only the |opt.mongodb-metrics| service:

  .. include:: .res/code/pmm-admin.start.mongodb-metrics.txt
		
For more information, run
|pmm-admin.start|
|opt.help|.

.. _pmm-admin.stop:

`Stopping monitoring services <pmm-admin.html#pmm-admin-stop>`_
================================================================================

Use the |pmm-admin.stop| command to stop services
managed by this |pmm-client|.

.. _pmm-admin.stop.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.stop.service-name.options:

.. include:: .res/code/pmm-admin.stop.service.name.options.txt

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. _pmm-admin.stop.options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.stop| command:

|opt.all|
  Stop all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. _pmm-admin.stop.services:

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to stop.
To see which services are available, run |pmm-admin.list|_.

.. _pmm-admin.stop.examples:

.. rubric:: EXAMPLES

* To stop all available services for this |pmm-client|:

  .. include:: .res/code/pmm-admin.stop.all.txt
		
* To stop all services related to |mysql|:

  .. include:: .res/code/pmm-admin.stop.mysql.txt
		   
* To stop only the |opt.mongodb-metrics| service:

  .. include:: .res/code/pmm-admin.stop.mongodb-metrics.txt
		   
For more information, run
|pmm-admin.stop|
|opt.help|.

.. _pmm-admin.uninstall:

`Cleaning Up Before Uninstall <pmm-admin.html#pmm-admin-uninstall>`_
================================================================================

Use the |pmm-admin.uninstall| command to remove all services even if
|pmm-server| is not available.  To uninstall |pmm| correctly, you first need to
remove all services, then uninstall |pmm-client|, and then stop and remove
|pmm-server|.  However, if |pmm-server| is not available (disconnected or shut
down), |pmm-admin.rm|_ will not work.  In this case, you can use
|pmm-admin.uninstall| to force the removal of monitoring services enabled for
|pmm-client|.

.. note:: Information about services will remain in |pmm-server|, and it will
   not let you add those services again.  To remove information about orphaned
   services from |pmm-server|, once it is back up and available to |pmm-client|,
   use the |pmm-admin.repair|_ command.

.. _pmm-admin.uninstall.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.uninstall.options:

.. include:: .res/code/pmm-admin.uninstall.options.txt

.. _pmm-admin.uninstall.options:

.. rubric:: OPTIONS

The |pmm-admin.uninstall| command does not have its own options, but you can use
:ref:`global options that apply to any other command <pmm-admin.options>`.

For more information, run
|pmm-admin.uninstall|
|opt.help|.

.. _pmm-admin.service-aliases:

`Monitoring Service Aliases <pmm-admin.html#pmm-admin-service-aliases>`_
================================================================================

The following aliases are used to designate PMM services that you want to
:ref:`add <pmm-admin.add>`, :ref:`remove <pmm-admin.rm>`, :ref:`restart
<pmm-admin.restart>`, :ref:`start <pmm-admin.start>`, or :ref:`stop
<pmm-admin.stop>`:

.. _code.pmm-admin.uninstall.alias-services:

.. include:: .res/table/alias.services.txt

.. include:: .res/replace.txt

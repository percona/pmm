.. _pmm-admin:

================================================================================
Managing |pmm-client|
================================================================================

Use the |pmm-admin| tool to manage |pmm-client|.

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

.. replacements moved to names.txt

.. _pmm-admin.add:

Adding monitoring services
================================================================================

Use the |pmm-admin.add| command to add monitoring services.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add [OPTIONS] [SERVICE]

.. _pmm-admin.add-options:

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.add| command:

|opt.dev-enable|
  Enable experimental features.

|opt.service-port|

  Specify the :ref:`service port <service-port>`.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`,
along with any relevant additional arguments.

For more information, run
|pmm-admin.add|
|opt.help|.

.. _pmm/pmm-admin/external-monitoring-service.adding:

Adding external monitoring services
--------------------------------------------------------------------------------

The |pmm-admin.add| command is also used to add external :term:`monitoring
services <External Monitoring Service>`. This command adds an external
monitoring service assuming that the underlying |prometheus| exporter is
already set up and accessible.

To add an external monitoring service use the |opt.external-metrics| service
followed by the name of a |prometheus| job, URL and port number to reach it.

.. code-block:: text

   pmm-admin add external:metrics JOB-NAME URL:PORT-NUMBER

The following example adds an external monitoring service which monitors a
|postgresql| instance at 192.168.200.1, port 9187

.. code-block:: text

   pmm-admin add external:metrics postgresql 192.168.200.1:9187

If the command succeeds then running :ref:`pmm-admin.list` shows the
newly added external exporter at the bottom of the command's output:

.. code-block:: text

   PMM Server      | 192.168.100.1
   Client Name     | percona
   Client Address  | 192.168.200.1
   Service Manager | linux-systemd

   -------------- -------- ----------- -------- ------------ --------
   SERVICE TYPE   NAME     LOCAL PORT  RUNNING  DATA SOURCE  OPTIONS 
   -------------- -------- ----------- -------- ------------ --------
   linux:metrics  percona  42000       YES                 -                    


   Name      Scrape interval  Scrape timeout  Metrics path  Scheme  Instances
   postgres  1s               1s              /metrics      http    192.168.200.1:9187


.. _pmm.pmm-admin.monitoring-service.pass-parameter:

Passing parameters to a monitoring service
--------------------------------------------------------------------------------

|pmm-admin.add| sends all options which follow :option:`--` (two
consecutive dashes delimited by whitespace) to the monitoring service as
parameters.

.. code-block:: bash
   :caption: Passing :option:`--collect.perf_schema.eventsstatements` to the
             |opt.mysql-metrics| monitoring service
   :name: pmm.pmm-admin.monitoring-service.pass-parameter.example

   $ sudo pmm-admin add mysql:metrics -- --collect.perf_schema.eventsstatements

.. code-block:: bash
   :caption: Passing :option:`--collect.perf_schema.eventswaits=false` to the
             :option:`mysql:metrics` monitoring service
   :name: pmm.pmm-admin.monitoring-service.pass-parameter.example2

   $ sudo pmm-admin add mysql:metrics -- --collect.perf_schema.eventswaits=false


.. _pmm.pmm-admin.mongodb.pass-ssl-parameter:

Passing SSL parameters to the mongodb monitoring service
--------------------------------------------------------------------------------

SSL/TLS related parameters are passed to an SSL enabled MongoDB server as
monitoring service parameters along with the |pmm-admin.add| command
when adding the |opt.mongodb-queries| monitoring service.

.. code-block:: bash
   :caption: Passing an SSL/TLS parameter to |mongod| to enables
             a TLS connection.

   $ sudo pmm-admin add mongodb:queries -- --mongodb.tls

.. list-table:: Supported SSL/TLS Parameters
   :widths: 25 75
   :header-rows: 1

   * - Parameter
     - Description
   * - :option:`--mongodb.tls`
     - Enable a TLS connection with mongo server
   * - :option:`--mongodb.tls-ca` *string*
     - A path to a PEM file that contains the CAs that are trusted for server connections.
       *If provided*: MongoDB servers connecting to should present a certificate signed by one of this CAs.
       *If not provided*: System default CAs are used.
   * - :option:`--mongodb.tls-cert` *string*
     - A path to PEM file that contains the certificate (and optionally also the private key in PEM format).
       This should include the whole certificate chain.
       *If provided*: The connection will be opened via TLS to the MongoDB server.
   * - :option:`--mongodb.tls-disable-hostname-validation`
     - Do hostname validation for the server connection.
   * - :option:`--mongodb.tls-private-key` *string*
     - A path to a PEM file that contains the private key (if not contained in the :option:`mongodb.tls-cert` file).


.. _pmm-admin-add-linux-metrics:

Adding general system metrics service
--------------------------------------------------------------------------------

Use the |opt.linux-metrics| alias to enable general system metrics monitoring.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add linux:metrics [NAME] [OPTIONS]

This creates the ``pmm-linux-metrics-42000`` service
that collects local system metrics for this particular OS instance.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following option can be used with the ``linux:metrics`` alias:

``--force``
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

.. _pmm-admin.add-mysql-queries:

Adding |mysql| query analytics service
--------------------------------------------------------------------------------

Use the |opt.mysql-queries| alias to enable MySQL query analytics.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mysql:queries [NAME] [OPTIONS]

This creates the ``pmm-mysql-queries-0`` service
that is able to collect QAN data for multiple remote MySQL server instances.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following options can be used with the ``mysql:queries`` alias:

``--create-user``
  Create a dedicated MySQL user for |pmm-client| (named ``pmm``).

``--create-user-maxconn``
  Specify maximum connections for the dedicated MySQL user (default is 10).

``--create-user-password``
  Specify password for the dedicated MySQL user.

``--defaults-file``
  Specify path to :file:`my.cnf`.

``--disable-queryexamples``
  Disable collection of query examples.

``--force``
  Force to create or update the dedicated MySQL user.

``--host``
  Specify the MySQL host name.

``--password``
  Specify the password for MySQL user with admin privileges.

``--port``
  Specify the MySQL instance port.

``--query-source``
  Specify the source of data:

  * ``auto``: Select automatically (default).
  * ``slowlog``: Use the slow query log.
  * ``perfschema``: Use Performance Schema.

``--socket``
  Specify the MySQL instance socket file.

``--user``
  Specify the name of MySQL user with admin privileges.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

.. rubric:: DETAILED DESCRIPTION

When adding the |mysql| query analytics service, the |pmm-admin| tool
will attempt to automatically detect the local |mysql| instance and
|mysql| superuser credentials.  You can use options to provide this
information, if it cannot be detected automatically.

You can also specify the |opt.create-user| option to create a dedicated
``pmm`` user on the MySQL instance that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the MySQL superuser.

For example, to set up remote monitoring of QAN data on a |mysql| server
located at 192.168.200.2, use a command similar to the following:

.. code-block:: bash

   sudo pmm-admin add mysql:queries --user root --password root --host 192.168.200.2 --create-user

QAN can use either the slow query log or Performance Schema as the source.
By default, it chooses the slow query log for a local MySQL instance
and Performance Schema otherwise.
For more information about the differences, see :ref:`perf-schema`.

You can explicitely set the query source when adding a QAN instance
using the ``--query-source`` option.

For more information, run
|pmm-admin.add|
|opt.mysql-queries|
|opt.help|.

.. _pmm-admin.add-mysql-metrics:

Adding |mysql| metrics service
--------------------------------------------------------------------------------

Use the |opt.mysql-metrics| alias to enable MySQL metrics monitoring.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mysql:metrics [NAME] [OPTIONS]

This creates the ``pmm-mysql-metrics-42002`` service
that collects MySQL instance metrics.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following options can be used with the |opt.mysql-metrics| alias:

|opt.create-user|
  Create a dedicated |mysql| user for |pmm-client| (named ``pmm``).

``--create-user-maxconn``
  Specify maximum connections for the dedicated |mysql| user (default is 10).

``--create-user-password``
  Specify password for the dedicated |mysql| user.

``--defaults-file``
  Specify the path to :file:`my.cnf`.

``--disable-binlogstats``
  Disable collection of binary log statistics.

``--disable-processlist``
  Disable collection of process state metrics.

``--disable-tablestats``
  Disable collection of table statistics.

``--disable-tablestats-limit``
  Specify the maximum number of tables
  for which collection of table statistics is enabled
  (by default, the limit is 1 000 tables).

``--disable-userstats``
  Disable collection of user statistics.

``--force``
  Force to create or update the dedicated |mysql| user.

``--host``
  Specify the |mysql| host name.

``--password``
  Specify the password for |mysql| user with admin privileges.

``--port``
  Specify the |mysql| instance port.

``--socket``
  Specify the |mysql| instance socket file.

``--user``
  Specify the name of |mysql| user with admin privileges.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

.. rubric:: DETAILED DESCRIPTION

When adding the |mysql| metrics monitoring service,
the |pmm-admin| tool attempts to automatically detect
the local |mysql| instance and |mysql| superuser credentials.
You can use options to provide this information,
if it cannot be detected automatically.

You can also specify the |opt.create-user| option to create a dedicated
``pmm`` user on the |mysql| host that you want to monitor.
This user will be given all the necessary privileges for monitoring,
and is recommended over using the |mysql| superuser.

For example,
to set up remote monitoring of |mysql| metrics
on a server located at 192.168.200.3,
use a command similar to the following:

.. code-block:: bash

   sudo pmm-admin add mysql:metrics --user root --password root --host 192.168.200.3 --create-user

For more information, run
|pmm-admin.add|
|opt.mysql-metrics|
|opt.help|.

.. _pmm-admin.add-mongodb-queries:

Adding |mongodb| query analytics service
--------------------------------------------------------------------------------


Use the |opt.mongodb-queries| alias to enable MongoDB query analytics.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mongodb:queries [NAME] [OPTIONS]

This creates the ``pmm-mongodb-queries-0`` service
that is able to collect QAN data for multiple remote MongoDB server instances.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following options can be used with the ``mongodb:queries`` alias:

|opt.uri|
  Specify the |mongodb| instance URI with the following format::

   [mongodb://][user:pass@]host[:port][/database][?options]

  By default, it is ``localhost:27017``.

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`,
as well as
:ref:`options that apply to adding services in general <pmm-admin.add-options>`.

For more information, run
|pmm-admin.add|
|opt.mongodb-queries|
|opt.help|.

.. _pmm-admin.add.mongodb-metrics:

Adding |mongodb| metrics service
--------------------------------------------------------------------------------

Use the |opt.mongodb-metrics| alias to enable MongoDB metrics monitoring.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add mongodb:metrics [NAME] [OPTIONS]

This creates the ``pmm-mongodb-metrics-42003`` service
that collects local MongoDB metrics for this particular MongoDB instance.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following options can be used with the |opt.mongodb-metrics| alias:

``--cluster``
  Specify the MongoDB cluster name.

``--uri``
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

.. _pmm-admin.add-proxysql-metrics:

Adding ProxySQL metrics service
--------------------------------------------------------------------------------

Use the |opt.proxysql-metrics| alias
to enable ProxySQL performance metrics monitoring.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin add proxysql:metrics [NAME] [OPTIONS]

This creates the ``pmm-proxysql-metrics-42004`` service
that collects local ProxySQL performance metrics.

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following option can be used with the |opt.proxysql-metrics| alias:

``--dsn``
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

.. _pmm-admin.check-network:

Checking network connectivity
================================================================================

Use the |pmm-admin.check-network| command to run tests
that verify connectivity between |pmm-client| and |pmm-server|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin check-network [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.check-network| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. rubric:: DETAILED DESCRIPTION

Connection tests are performed both ways,
with results separated accordingly:

* ``Client --> Server``

  Pings Consul API, Query Analytics API, and Prometheus API
  to make sure they are alive and reachable.

  Performs a connection performance test to see the latency
  from |pmm-client| to |pmm-server|.

* ``Client <-- Server``

  Checks the status of Prometheus endpoints
  and makes sure it can scrape metrics from corresponding exporters.

  Successful pings of |pmm-server| from |pmm-client|
  do not mean that Prometheus is able to scrape from exporters.
  If the output shows some endpoints in problem state,
  make sure that the corresponding service is running
  (see |pmm-admin.list|_).
  If the services that correspond to problematic endpoints are running,
  make sure that firewall settings on the |pmm-client| host
  allow incoming connections for corresponding ports.

.. rubric:: OUTPUT EXAMPLE

.. code-block:: text
   :emphasize-lines: 1

   $ sudo pmm-admin check-network
   PMM Network Status

   Server Address | 192.168.100.1
   Client Address | 192.168.200.1

   * System Time
   NTP Server (0.pool.ntp.org)         | 2017-05-03 12:05:38 -0400 EDT
   PMM Server                          | 2017-05-03 16:05:38 +0000 GMT
   PMM Client                          | 2017-05-03 12:05:38 -0400 EDT
   PMM Server Time Drift               | OK
   PMM Client Time Drift               | OK
   PMM Client to PMM Server Time Drift | OK

   * Connection: Client --> Server
   -------------------- -------------
   SERVER SERVICE       STATUS
   -------------------- -------------
   Consul API           OK
   Prometheus API       OK
   Query Analytics API  OK

   Connection duration | 166.689µs
   Request duration    | 364.527µs
   Full round trip     | 531.216µs

   * Connection: Client <-- Server
   ---------------- ----------- -------------------- -------- ---------- ---------
   SERVICE TYPE     NAME        REMOTE ENDPOINT      STATUS   HTTPS/TLS  PASSWORD
   ---------------- ----------- -------------------- -------- ---------- ---------
   linux:metrics    mongo-main  192.168.200.1:42000  OK       YES        -
   mongodb:metrics  mongo-main  192.168.200.1:42003  PROBLEM  YES        -

For more information, run
|pmm-admin.check-network|
|opt.help|.

.. _pmm-admin.config:

Configuring PMM Client
================================================================================

Use the |pmm-admin.config| command to configure
how |pmm-client| communicates with |pmm-server|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin config [OPTIONS]

.. rubric:: OPTIONS

The following options can be used with the |pmm-admin.config| command:

``--bind-address``
  Specify the bind address,
  which is also the local (private) address
  mapped from client address via NAT or port forwarding
  By default, it is set to the client address.

``--client-address``
  Specify the client address,
  which is also the remote (public) address for this system.
  By default, it is automatically detected via request to server.

``--client-name``
  Specify the client name.
  By default, it is set to the host name.

``--force``
  Force to set the client name on initial setup
  after uninstall with unreachable server.

``--server``
  Specify the address of the |pmm-server| host.
  If necessary, you can also specify the port after colon, for example::

   pmm-admin config --server 192.168.100.6:8080

  By default, port 80 is used with SSL disabled,
  and port 443 when SSL is enabled.

``--server-insecure-ssl``
  Enable insecure SSL (self-signed certificate).

``--server-password``
  Specify the HTTP password configured on |pmm-server|.

``--server-ssl``
  Enable SSL encryption for connection to |pmm-server|.

``--server-user``
  Specify the HTTP user configured on |pmm-server| (default is ``pmm``).

You can also use
:ref:`global options that apply to any other command <pmm-admin.options>`.

For more information, run |pmm-admin.config| --help.

.. _pmm-admin.help:

Getting help for any command
================================================================================

Use the |pmm-admin.help| command to print help for any command.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin help [COMMAND]

This will print help information and exit.  The actual command is not run
and options are ignored.

.. note:: You can also use the global ``-h`` or ``--help`` option
   after any command to get the same help information.

.. rubric:: COMMANDS

You can print help information for any :ref:`command <pmm-admin.commands>`
or :ref:`service alias <pmm-admin.service-aliases>`.

.. _pmm-admin.info:

Getting information about PMM Client
================================================================================

Use the |pmm-admin.info| command
to print basic information about |pmm-client|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin info [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.info| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. rubric:: OUTPUT

The output provides the following information:

* Version of |pmm-admin|
* |pmm-server| host address, and local host name and address
  (this can be configured using |pmm-admin.config|_)
* System manager that |pmm-admin| uses to manage PMM services
* Go version and runtime information

For example:

.. code-block:: text
   :emphasize-lines: 1

   $ sudo pmm-admin info

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   Go Version      | 1.8
   Runtime Info    | linux/amd64

For more information, run
|pmm-admin.info|
|opt.help|.

.. _pmm-admin.list:

Listing monitoring services
================================================================================

Use the |pmm-admin.list| command to list all enabled services with details.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin list [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.list| command supports :ref:`global options that apply to any other command
<pmm-admin.options>` and also provides a machine friendly JSON output.

``--json``
   list the enabled services as a JSON document. The information provided in the
   standard tabular form is captured as keys and values. The general information
   about the computer where this pmm-client is installed is given as top level
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
   ``Err`` top level element in the JSON output. Similarly, the ``ExternalErr`` element
   reports errors in external services.

   The ``Services`` top level element contains a list of documents which represent enabled
   monitoring services. Each attribute in a document maps to the column in the tabular
   output.

   The ``ExternalServices`` element contains a list of documents which represent
   enabled external monitoring services. Each attribute in a document maps to
   the column in the tabular output.

.. rubric:: OUTPUT

The output provides the following information:

* Version of |pmm-admin|
* |pmm-server| host address, and local host name and address
  (this can be configured using |pmm-admin.config|_)
* System manager that |pmm-admin| uses to manage PMM services
* A table that lists all services currently managed by ``pmm-admin``,
  with basic information about each service

For example, if you enable general OS and MongoDB metrics monitoring,
output should be similar to the following:

.. code-block:: text
   :emphasize-lines: 1

   $ pmm-admin list

   PMM Server      | 192.168.100.1
   Client Name     | ubuntu-amd64
   Client Address  | 192.168.200.1
   Service manager | linux-systemd

   ---------------- ----------- ----------- -------- ---------------- --------
   SERVICE TYPE     NAME        LOCAL PORT  RUNNING  DATA SOURCE      OPTIONS
   ---------------- ----------- ----------- -------- ---------------- --------
   linux:metrics    mongo-main  42000       YES      -
   mongodb:metrics  mongo-main  42003       YES      localhost:27017


.. _pmm-admin.ping:

Pinging |pmm-server|
================================================================================

Use the |pmm-admin.ping| command to verify connectivity with |pmm-server|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin ping [OPTIONS]

If the ping is successful, it returns ``OK``.

.. rubric:: OPTIONS

The |pmm-admin.ping| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run
|pmm-admin.ping|
|opt.help|.

.. _pmm-admin.purge:

Purging metrics data
====================

Use the |pmm-admin.purge| command to purge metrics data
associated with a service on |pmm-server|.
This is usually required after you :ref:`remove a service <pmm-admin.rm>`
and do not want its metrics data to show up on graphs.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin purge [SERVICE [NAME]] [OPTIONS]

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. rubric:: OPTIONS

The |pmm-admin.purge| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

For more infomation, run
|pmm-admin.purge|
|opt.help|.

.. _pmm-admin.remove:
.. _pmm-admin.rm:

Removing monitoring services
================================================================================

Use the |pmm-admin.rm| command to remove monitoring services.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin rm [OPTIONS] [SERVICE]

When you remove a service,
collected data remains in Metrics Monitor on |pmm-server|.
To remove collected data, use the |pmm-admin.purge|_ command.

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.rm| command:

|opt.all|
  Remove all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`.
To see which services are enabled, run |pmm-admin.list|_.

.. rubric:: EXAMPLES

* To remove all services enabled for this |pmm-client|:

   .. code-block:: bash

      $ pmm-admin rm --all

* To remove all services related to MySQL:

   .. code-block:: bash

      $ pmm-admin rm mysql

* To remove only MongoDB metrics service:

   .. code-block:: bash

      $ pmm-admin rm mongodb:metrics

For more information, run |pmm-admin.rm| --help.

.. _pmm-admin.repair:

Removing orphaned services
================================================================================

Use the |pmm-admin.repair| command
to remove information about orphaned services from |pmm-server|.
This can happen if you removed services locally
while |pmm-server| was not available (disconnected or shut down),
for example, using the |pmm-admin.uninstall|_ command.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin repair [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.repair| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run |pmm-admin.repair| --help.

.. _pmm-admin.restart:

Restarting monitoring services
==============================

Use the |pmm-admin.restart| command to restart services
managed by this |pmm-client|.
This is the same as running |pmm-admin.stop|_ and |pmm-admin.start|_.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin restart [SERVICE [NAME]] [OPTIONS]

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.restart| command:

|opt.all|
  Restart all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to restart.
To see which services are available, run |pmm-admin.list|_.

.. rubric:: EXAMPLES

* To restart all available services for this |pmm-client|:

   .. code-block:: bash

      sudo pmm-admin restart --all

* To restart all services related to MySQL:

   .. code-block:: bash

      sudo pmm-admin restart mysql

* To restart only MongoDB metrics service:

   .. code-block:: bash

      sudo pmm-admin restart mongodb:metrics

For more information, run |pmm-admin.restart| :option:`--help`.

.. _pmm-admin.show-passwords:

Getting passwords used by PMM Client
================================================================================

Use the |pmm-admin.show-passwords| command to print credentials
stored in the configuration file
(by default: :file:`/usr/local/percona/pmm-client/pmm.yml`).

.. rubric:: USAGE

.. code-block:: text

   pmm-admin show-passwords [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.show-passwords| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. rubric:: OUTPUT

This command prints HTTP authentication credentials
and the password for the ``pmm`` user that is created on the MySQL instance
if you specify the |opt.create-user| option
when :ref:`adding a service <pmm-admin.add>`.

.. code-block:: bash
   :emphasize-lines: 1

   $ sudo pmm-admin show-passwords
   HTTP basic authentication
   User     | aname
   Password | secr3tPASS

   MySQL new user creation
   Password | g,3i-QR50tQJi9M1yl9-

For more information, run |pmm-admin.show-passwords|  |opt.help|.

.. _pmm-admin.start:

Starting monitoring services
================================================================================

Use the |pmm-admin.start| command to start services
managed by this |pmm-client|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin start [SERVICE [NAME]] [OPTIONS]

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.start| command:

|opt.all|
  Start all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to start.
To see which services are available, run |pmm-admin.list|_.

.. rubric:: EXAMPLES

* To start all available services for this |pmm-client|:

   .. code-block:: bash

      sudo pmm-admin start --all

* To start all services related to |mysql|:

   .. code-block:: bash

      sudo pmm-admin start mysql

* To start only MongoDB metrics service:

   .. code-block:: bash

      sudo pmm-admin start mongodb:metrics

For more information, run
|pmm-admin.start|
|opt.help|.

.. _pmm-admin.stop:

Stopping monitoring services
================================================================================

Use the |pmm-admin.stop| command to stop services
managed by this |pmm-client|.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin stop [SERVICE [NAME]] [OPTIONS]

.. note:: It should be able to detect the local |pmm-client| name,
   but you can also specify it explicitly as an argument.

.. rubric:: OPTIONS

The following option can be used with the |pmm-admin.stop| command:

|opt.all|
  Stop all monitoring services.

You can also use
:ref:`global options that apply to any other command
<pmm-admin.options>`.

.. rubric:: SERVICES

Specify a :ref:`monitoring service alias <pmm-admin.service-aliases>`
that you want to stop.
To see which services are available, run |pmm-admin.list|_.

.. rubric:: EXAMPLES

* To stop all available services for this |pmm-client|:

   .. code-block:: bash

      sudo pmm-admin stop --all

* To stop all services related to MySQL:

   .. code-block:: bash

      sudo pmm-admin stop mysql

* To stop only MongoDB metrics service:

   .. code-block:: bash

      sudo pmm-admin stop mongodb:metrics

For more information, run |pmm-admin.stop|  |opt.help|.

.. _pmm-admin.uninstall:

Cleaning up PMM Client before uninstall
================================================================================

Use the |pmm-admin.uninstall| command to remove all services even if
|pmm-server| is not available.  To uninstall |pmm| correctly, you
first need to remove all services, then uninstall |pmm-client|, and
then stop and remove |pmm-server|.  However, if |pmm-server| is not
available (disconnected or shut down), |pmm-admin.rm|_ will not work.
In this case, you can use |pmm-admin.uninstall| to force the removal
of monitoring services enabled for |pmm-client|.

.. note:: Information about services will remain in |pmm-server|,
   and it will not let you add those services again.
   To remove information about orphaned services from |pmm-server|,
   once it is back up and available to |pmm-client|,
   use the |pmm-admin.repair|_ command.

.. rubric:: USAGE

.. code-block:: text

   pmm-admin uninstall [OPTIONS]

.. rubric:: OPTIONS

The |pmm-admin.uninstall| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`.

For more information, run
|pmm-admin.uninstall|
|opt.help|.

.. _pmm-admin.service-aliases:

Monitoring Service Aliases
================================================================================

The following aliases are used to designate PMM services
that you want to :ref:`add <pmm-admin.add>`, :ref:`remove <pmm-admin.rm>`,
:ref:`restart <pmm-admin.restart>`,
:ref:`start <pmm-admin.start>`, or :ref:`stop <pmm-admin.stop>`:

.. list-table::
   :widths: 25 75
   :header-rows: 1

   * - Alias
     - Services

   * - |opt.linux-metrics|
     - General system metrics monitoring service

   * - |opt.mysql-metrics|
     - |mysql| metrics monitoring service

   * - |opt.mysql-queries|
     - |mysql| query analytics service

   * - |opt.mongodb-metrics|
     - |mongodb| metrics monitoring service

   * - |opt.mongodb-queries|
     - |mongodb| query analytics service

   * - |opt.proxysql-metrics|
     - |proxysql| metrics monitoring service

   * - |opt.mysql|
     - Complete |mysql| instance monitoring:

       * |opt.linux-metrics|
       * |opt.mysql-metrics|
       * |opt.mysql-queries|

   * - |opt.mongodb|
     - Complete |mongodb| instance monitoring:

       * |opt.linux-metrics|
       * |opt.mongodb-metrics|
       * |opt.mongodb-queries|

.. include:: .resources/name.txt

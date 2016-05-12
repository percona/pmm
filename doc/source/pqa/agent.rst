.. _agent:

=============================
Percona Query Analytics Agent
=============================

PQA Agent is the client-side tool for collecting and sending MySQL performance data to a :ref:`PQA Datastore <datastore>` over a secure websocket connection. It uses either the MySQL slow log or Performance Schema.

.. contents::
   :local:

Installation
============

PQA Agent is a static binary with no external dependencies. You can install it on the same host as MySQL or configure it to collect query data remotely.

.. _agent-reqs:

Requirements
------------

* **Linux**: The agent is tested on latest Debian, Ubuntu, CentOS, and Red Hat Enterprise Linux distributions.

  .. note:: The agent must run as *root*, because MySQL slow log file has restricted permissions.

* **MySQL**: The agent requires MySQL 5.1 or later for collecting data from the slow log, and MySQL 5.6.9 or later for Performance Schema.

* **Network**: Outbound websocket connection to a :ref:`PQA Datastore <datastore>` (port 9001 by default).

The PQA Agent installer uses :command:`mysql --print-defaults`
to detect local MySQL instance and MySQL superuser credentials.
Make sure that the necessary options are specified in :file:`~/.my.cnf`
(for root). For example:

.. code-block:: none

   user=root
   password=pass
   socket=/var/run/mysqld/mysqld.sock

MySQL superuser credentials are used to create a MySQL user for PQA Agent
with the following privileges:

* ``SUPER, PROCESS, USAGE, SELECT ON *.* TO 'qan-agent'@'localhost'``
* ``UPDATE, DELETE, DROP ON performance_schema.* TO 'qan-agent'@'localhost'``

.. note:: Instead of ``localhost``, a specific IP (such as ``127.0.0.1``)
   or the ``%`` wildcard can be used.

Quick Install
-------------

1. Download the archive with PQA Agent from https://www.percona.com/redir/downloads/TESTING/ppl/open-source/ppl-agent.tar.gz

2. Extract the archive and change to the extracted directory.

3. Run the :file:`./install` script as root, specifying the :ref:`PQA Datastore <datastore>` host name or IP address. For example, if the datastore is running locally:

   .. code-block:: bash

      $ sudo ./install 127.0.0.1

If the installer is not able to detect and access the MySQL instance with proper privileges, you can specify the MySQL superuser credentials explicitely. You can also specify a port number if it is not the default 9001. For example:

.. code-block:: bash

   $ sudo ./install -user=root -pass=password 192.168.1.10:9123

In the example above, ``root`` is the MySQL user with grant privileges that can be used by the install script to create a PQA Agent user in MySQL.

For more information about install script options, see :ref:`install-options`.

.. _install-options:

Installation Options
--------------------

The PQA Agent install script has the following syntax:

.. code-block:: none

   install [OPTIONS] <DATASTORE_HOST>[:PORT]

The following options are available:

.. default-domain:: none

.. option:: -agent-password <password>

   Specifies existing PQA Agent MySQL user password

.. option:: -agent-user <name>

   Specifies existing PQA Agent MySQL user name

.. option:: -basedir <dir>

   Specifies the PQA Agent installation directory (default is :file:`/usr/local/percona/agent`)

.. option:: -debug

   Enables debug

.. option:: -defaults-file <path>

   Specifies the path to :file:`my.cnf` (by default, it is located in the home directory of the user running MySQL: :file:`~/my.cnf`)

.. option:: -help -h -?

   Print help information and exit

.. option:: -host <host>

   Specifies the MySQL host name or IP address (by default, it assumes a local MySQL host)

.. option:: -max-user-connections <integer>

   Specifies the maximum number of MySQL connections (default is 5)

.. option:: -old-passwords

   Enables MySQL old passwords

.. option:: -password <password>

   Specifies MySQL superuser password

.. option:: -port <port>

   Specifies MySQL port

.. option:: -query-source <source>

   Specifies where to collect queries. The following values are possible:

   * ``slowlog``: Slow query log
   * ``perfschema``: Performance Schema
   * ``auto``: Automatically select source (default)

.. option:: -socket <file>

   Specifies the MySQL socket file (default is :file:`/var/run/mysqld/mysqld.sock`)

.. option:: -user <name>

   Specifies MySQL superuser name

.. option:: -uninstall

   Instructs to stop the agent and uninstall it

Logging
=======

PQA Agent has two logging systems:

* **Online**: Sends all log messages (except debug entries) to :ref:`PQA Datastore <datastore>`.

* **Offline**: Writes log messages to ``STDOUT`` (info entries) and ``STDERR`` (warning, error, and fatal entries).

You can use the :file:`config/log.conf` file to set the log level and disable online logging. The following examples show which log entries are written where depending on the log configuration. For more information about the log config, see :ref:`log-config`.

**Default**

No ``config/log.conf`` file or it contains ``{"Level":"warning","Offline":"false"}``:

==========  ========= ====== ======
Level       Datastore STDOUT STDERR
==========  ========= ====== ======
Debug
Info        *
Warning     *                *
Error       *                *
Fatal       *                *
==========  ========= ====== ======

**Traditional**

If ``config/log.conf`` contains ``{"Level":"info","Offline":"true"}``:

==========  ========= ====== ======
Level       Datastore STDOUT STDERR
==========  ========= ====== ======
Debug
Info                  *
Warning                      *
Error                        *
Fatal                        *
==========  ========= ====== ======

Configuration
=============

The PQA Agent is designed to be configured through the :ref:`PQA App <webapp>`, but it is possible to manually edit the config files. Please keep the following in mind:

* The :file:`config/` directory has full permissions only for the owner (root). No other user can read, write, or execute any of the config files.

* All config files are strict JSON (that means no trailling commas).

* PQA Agent must be restarted after changing a config file manually.

* When you configure PQA Agent via the :ref:`PQA App <webapp>`, config files are changed on the fly (no restart is required).

* Boolean values are strings (fuzzy bools): ``true``, ``yes``, and ``on`` mean true; anything else means false.

* Only empty values are omitted:

  * There is no empty value for boolean

  * Empty value for strings is ``""``

  * For numeric values, ``0`` is considered *not set* and the default value is used

General Agent Configuration
---------------------------

The :file:`agent.conf` file is the only required config file. The folllowing variables are available:

.. describe:: ApiHostname

   :Required: Yes
   :Type: String
   :Default: none
   :Purpose: Specifies the :ref:`PQA Datastore <datastore>` host name or IP address and port
   :Example: ``127.0.0.1:9001``

.. describe:: Keepalive

   :Required: No
   :Type: Integer
   :Default: ``76``
   :Purpose: Specifies how often to ping :ref:`PQA Datastore <datastore>` (in seconds)

.. describe:: Links

   :Required: No
   :Type: String
   :Default: none
   :Purpose: Specifies API links sent by :ref:`PQA Datastore <datastore>`. Do not change!

.. describe:: PidFile

   :Required: No
   :Type: String
   :Default: ``percona-qan-agent.pid``
   :Purpose: Specifies the PID file relative to base installation directory

.. describe:: UUID

   :Required: Yes
   :Type: String
   :Default: none
   :Purpose: Specifies the unique identifier of the agent instance
   :Example: ``d64661aa05e249ff61eb6e85507f904c``

Data Processing Configuration
-----------------------------

The :file:`data.conf` file is optional. The following variables are available:

.. describe:: Blackhole

   :Required: No
   :Type: Boolean
   :Default: ``false``
   :Purpose: Specifies whether to send data to :file:`/dev/null` instead of :ref:`PQA Datastore <datastore>`

.. describe:: Encoding

   :Required: No
   :Type: String
   :Default: ``gzip``
   :Purpose: Specifies encoding method
   :Values: * ``gzip``: Encode using :command:`gzip`
            * ``none``: Do not encode

.. describe:: Limits

   :Required: No
   :Type: Subdocument
   :Purpose: Limits the size of the data spool
   :Variables: .. describe:: MaxAge

               :Required: No
               :Type: Integer
               :Default: ``86400`` (one day)
               :Purpose: Specifies maximum age of data files to keep (in seconds). Older files are purged.

            .. describe:: MaxFiles

               :Required: No
               :Type: Integer
               :Default: ``1000``
               :Purpose: Specifies maximum number of files to keep. When the spool has more files, the oldest are purged.

            .. describe:: MaxSize

               :Required: No
               :Type: Integer
               :Default: ``104857600`` (100 MiB)
               :Purpose: Specifies the maximum total size of files to keep. When the spool becomes larger, the oldest files are purged.

.. describe:: SendInterval

   :Required: No
   :Type: Integer
   :Default: ``63``
   :Purpose: Specifies how often to send data to :ref:`PQA Datastore <datastore>` (in seconds)

.. _log-config:

Logging Configuration
---------------------

The :file:`log.conf` file is optional. The following variables are available:

.. describe:: Level

   :Required: No
   :Type: String
   :Default: ``warning``
   :Purpose: Specifies the minimum level of detail for the offline log (affects only messages sent to ``STDOUT`` and ``STDERR``)
   :Values: * ``debug``: Log everything (use only for debugging)
            * ``info``: Log everything except debug entries
            * ``warning``: Log everything except debug and info entries
            * ``error``: Log all errors
            * ``fatal``: Log only critical errors

.. describe:: Offline

   :Required: No
   :Type: Boolean
   :Default: ``false``
   :Purpose: Specifies whether to disable online logging to :ref:`PQA Datastore <datastore>`

Query Analytics Configuration
-----------------------------

The PQA Agent can collect data from multiple MySQL instances, running a separate Query Analytics module for each. Parameters for a specific instance can be configured in the corresponding file :file:`qan-<UUID>.conf`, where the ``<UUID>`` suffix is the unique identifier of the MySQL instance. The following variables are available:

.. describe:: CollectFrom

   :Required: No
   :Type: String
   :Default: ``slowlog``
   :Purpose: Specifies the source for query data.
   :Values: * ``slowlog``: Slow query log
            * ``perfschema``: Performance Schema

.. describe:: ExampleQueries

   :Required: No
   :Type: Boolean
   :Default: ``true``
   :Purpose: Specifies whether to send an example for each query

.. describe:: Interval

   :Required: No
   :Type: Integer
   :Default: ``60``
   :Purpose: Specifies how often to collect and aggregate data (in seconds)

.. describe:: MaxSlowLogSize

   :Required: No
   :Type: Integer
   :Default: ``1073741824`` (1 GiB)
   :Purpose: Specifies the maximum allowed size for the slow log (in bytes). When the slow log becomes larger, it is rotated.

.. describe:: RemoveOldSlowLogs

   :Required: No
   :Type: Boolean
   :Default: ``true``
   :Purpose: Specifies whether to remove old slow log after rotating

.. describe:: ReportLimit

   :Required: No
   :Type: Integer
   :Default: ``200``
   :Purpose: Specifies how many queries sorted by total query time (per interval) to send to :ref:`QAN Datastore <datastore>`

.. describe:: Start

   :Required: No
   :Type: String
   :Purpose: Contains MySQL queries necessary to configure the server

.. describe:: Stop

   :Required: No
   :Type: String
   :Purpose: Contains MySQL queries necessary to un-configure the server

.. describe:: UUID

   :Required: Yes
   :Type: String
   :Default: None
   :Purpose: Specifies the unique identifier of the MySQL instance to which this config file applies (it matches the file name suffix)
   :Example: ``b2c28e3015b540494be4aa1c192b8a3c``

.. describe:: WorkerRunTime

   :Required: No
   :Type: Integer
   :Default: ``55``
   :Purpose: Specifies the maximum runtime for each worker per interval

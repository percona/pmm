.. _pmm.conf-mysql.settings.ps:

################################
Percona Server specific settings
################################

Not all dashboards in Metrics Monitor are available by default for all MySQL
variants and configurations: Oracle's MySQL, Percona Server. or MariaDB.
Some graphs require Percona Server, and specialized plugins, or additional
configuration.

.. toctree::
   :maxdepth: 2

   conf-mysql-ps-userstat
   conf-mysql-ps-qrt
   conf-mysql-ps-log-slow-rate-limit
   conf-mysql-ps-log-slow-verbosity
   conf-mysql-ps-slow-query-log-use-global-control

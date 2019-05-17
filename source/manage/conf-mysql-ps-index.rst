`Percona Server specific settings <pmm.conf-mysql.settings.ps>`_
==================================================================

Not all dashboards in |metrics-monitor| are available by default for all |mysql|
variants and configurations: |oracle|'s |mysql|, |percona-server|. or |mariadb|.
Some graphs require |percona-server|, and specialized plugins, or additional
configuration.

.. toctree::
   :maxdepth: 4

   conf-mysql-ps-userstat
   conf-mysql-ps-qrt
   conf-mysql-ps-log-slow-rate-limit
   conf-mysql-ps-log-slow-verbosity
   conf-mysql-ps-slow-query-log-use-global-control

.. include:: ../.res/replace.txt

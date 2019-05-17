--------------------------------------------------------------------------------
Adding a PostgreSQL host
--------------------------------------------------------------------------------

.. _pmm.qan.postgres.conf:

Understanding PostgreSQL metrics service
===============================================================================

Monitoring |postgresql| metrics with the `postgres_exporter <https://github.com/wrouesnel/postgres_exporter>`_ is enabled by ``pmm-admin add postgresql`` command. The ``postgresql`` alias will set up
``postgresql:metrics`` and also ``linux:metrics`` on a host (for more information, see `Adding monitoring services <https://www.percona.com/doc/percona-monitoring-and-management/pmm-admin.html#pmm-admin-add>`_).

``pmm-admin`` supports passing |postgresql| connection information via following flags:

==========================    =================================================
Flag                          Description 
==========================    =================================================
``--host``                    |postgresql| host
``--password``                |postgresql| password
``--port``                    |postgresql| port
``--user``                    |postgresql| user
==========================    =================================================

An example command line would look like this::

  pmm-admin add postgresql --host=localhost --password='secret' --port=5432 --user=pmm_user

.. note:: Capturing read and write time statistics is possible only if
   ``track_io_timing`` setting is enabled. This can be done either in
   configuration file or with the following query executed on the running
   system::

      ALTER SYSTEM SET track_io_timing=ON;
      SELECT pg_reload_conf();

.. _pmm.qan.postgres.conf.essential-permission.setting-up:

Setting Up the Required Permissions
--------------------------------------------------------------------------------

Percona recommends that a |postgresql| user be configured for ``SUPERUSER``
level access, in order to gather the maximum amount of data with a minimum
amount of complexity. This can be done with the following command for the
standalone |postgresql| installation::

  CREATE USER pmm_user WITH SUPERUSER ENCRYPTED PASSWORD 'secret';

.. note:: In case of monitoring a |postgresql| database running on
   an Amazon RDS instance, the command should look as follows::

      CREATE USER pmm_user WITH rds_superuser ENCRYPTED PASSWORD 'secret';

.. include:: ../.res/replace.txt

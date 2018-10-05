.. _pmm.qan.postgres.conf:

===============================================================================
Configuring |postgresql| for Monitoring
===============================================================================

Monitoring |postgresql| metrics with the `postgres_exporter <https://github.com/wrouesnel/postgres_exporter>`_ is enabled by ``pmm-admin add postgresql`` command. The ``postgresql`` alias will set up
``postgresql:metrics`` and also ``linux:metrics`` on a host (for more information, see `Adding monitoring services <https://www.percona.com/doc/percona-monitoring-and-management/pmm-admin.html#pmm-admin-add>`_).

``pmm-admin`` supports passing |postgresql| connection information via following flags:

==========================    =================================================
Flag                          Description 
==========================    =================================================
``--create-user``             create a new |postgresql| user (default: ``pmm``)
``--create-user-password``    optional password for a new PostgreSQL user
``--force``                   force user creation
``--host``                    |postgresql| host
``--password``                |postgresql| password
``--port``                    |postgresql| port
``--user``                    |postgresql| user
==========================    =================================================

..note: Password authentication should be turned on for the privileged
|postgresql| user (e.g. `postgres`)to make ``--create-user`` flag working.

An example command line would look like this::

  pmm-admin add postgresql --create-user --host=172.17.0.2 --password=ABC123 --port=5432 --user=postgres_exporter

Supported versions of PostgreSQL
--------------------------------

|pmm| follows `postgresql.org EOL policy <https://www.postgresql.org/support/versioning/>`_, and thus supports monitoring |postgresql| version 9.4 and up.  Older versions may work, but will not be supported.  For additional assistance, visit the Percona PMM Forums at https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management/.

.. _pmm.qan.postgres.conf.essential-permission.setting-up:

Setting Up the Required Permissions
================================================================================

User creation should follow these permissions::

   CREATE USER "pmm" WITH PASSWORD 'password';
   ALTER USER "pmm" SET SEARCH_PATH TO "pmm",pg_catalog;
   CREATE SCHEMA  "pmm" AUTHORIZATION "pmm";
   CREATE OR REPLACE VIEW "pmm".pg_stat_activity AS SELECT * from pg_catalog.pg_stat_activity;
   GRANT SELECT ON "pmm".pg_stat_activity TO "pmm";
   CREATE OR REPLACE VIEW "pmm".pg_stat_replication AS SELECT * from pg_catalog.pg_stat_replication;
   GRANT SELECT ON "pmm".pg_stat_replication TO "pmm";

.. include:: .res/replace.txt

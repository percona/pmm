.. _faq:

==========================
Frequently Asked Questions
==========================

Can I use PMM with Amazon RDS?
==============================

Yes, it is possible to monitor Amazon RDS or any remote MySQL instance.

First, create a MySQL user for PMM on the instance that you want to monitor
with the following privileges:

* ``SUPER, PROCESS, USAGE, SELECT ON *.* TO 'percona-qan-agent'@'localhost'``
* ``UPDATE, DELETE, DROP ON performance_schema.* TO 'percona-qan-agent'@'localhost'``

.. note:: Instead of ``localhost``, a specific IP (such as ``127.0.0.1``)
   or the ``%`` wildcard can be used.

Then enable data collection using the ``pmm-admin`` tool
on any host where PMM Client is installed.
Provide the superuser credentials and address of the RDS instance:

.. prompt:: bash

   pmm-admin -user <rds-user> -password <rds-pass> -host <rds-host> add mysql

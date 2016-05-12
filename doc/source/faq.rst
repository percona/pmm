.. _faq:

==========================
Frequently Asked Questions
==========================

Can I use PMM with Amazon RDS?
==============================

Yes, it is possible to monitor Amazon RDS or any remote MySQL instance.

First, create a MySQL user for PMM on the instance that you want to monitor,
and grant it ``UPDATE``, ``DELETE``, and ``DROP`` privileges
on ``performance_schema.*`` tables:

``GRANT UPDATE, DELETE, DROP ON performance_schema.* TO 'qan-agent'@'%'``

Then enable data collection using the ``pmm-admin`` tool
on any host where PMM Client is installed.
Provide the superuser credentials and address of the RDS instance,
and credentials of the MySQL user for PMM agent.
For example:

.. prompt:: bash

   pmm-admin -user root \
             -password pass \
             -host example.rds.amazonaws.com \
             -agent-user qan-agent \
             -agent-password qan-pass \
             add mysql

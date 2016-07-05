.. _faq:

==========================
Frequently Asked Questions
==========================

.. contents::
   :local:

Can I use PMM with Amazon RDS?
==============================

Yes, it is possible to monitor Amazon RDS or any remote MySQL instance.

First, create a MySQL user for PMM on the instance that you want to monitor,
and grant it ``UPDATE``, ``DELETE``, and ``DROP`` privileges
on ``performance_schema.*`` tables:

::

 mysql> GRANT UPDATE, DELETE, DROP ON performance_schema.* TO 'qan-agent'@'%'

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

How to control memory consumption for Prometheus?
=================================================

By default, Prometheus in PMM Server uses up to 256 MB of memory
for storing the most recently used data chunks.
Depending on the amount of data coming into Prometheus,
you may require a higher limit to avoid throttling data ingestion,
or allow less memory consumption if it is needed for other processes.

You can control the allowed memory consumption for Prometheus
by passing the ``METRICS_MEMORY`` environment variable
when creating and running the ``pmm-server`` container.
To set the environment variable, use the ``-e`` option.
The value must be passed in kilobytes.
For example, to set the limit to 4 GB of memory::

 -e METRICS_MEMORY=4194304

.. note:: The limit affects only memory reserved for data chunks.
   Actual RAM usage by Prometheus is higher.
   It is recommended to have at least three times more memory
   than the expected memory taken up by data chunks.

How to control data retention for Prometheus?
=============================================

By default, Prometheus in PMM Server stores time-series data for 8 days.
Depending on available disk space and your requirements,
you may need to adjust data retention time.

You can control data retention time for Prometheus
by passing the ``METRICS_RETENTION`` environment variable
when creating and running the ``pmm-server`` container.
To set the environment variable, use the ``-e`` option.
The value is passed as a combination of hours, minutes, and seconds.
For example, the default value of 8 days is ``192h0m0s``.
You probably do not need to be more precise than the number hours,
so you can discard the minutes and seconds.
For example, to set the retention to 30 days::

 -e METRICS_RETENTION=720h


.. _amazon-rds:

=========================
Using PMM with Amazon RDS
=========================

It is possible to use PMM for monitoring Amazon RDS
(just like any remote MySQL instance).

First of all, ensure that there is minimal latency between *PMM Server*
and the RDS instance.
Network connectivity can become an issue for Prometheus to scrape metrics
with 1 second resolution.
We strongly suggest that you run *PMM Server* on AWS.

.. note:: If latency is higher than 1 second,
   you should change the minimum resolution
   by setting the ``METRICS_RESOLUTION`` environment variable
   when :ref:`creating and running the PMM Server container <server-container>`.
   For more information, see :ref:`metrics-resolution`.

Query analytics requires :ref:`perf-schema` as the query source.
Enable the ``performance_schema`` option under **Parameter Groups** on RDS 
(you will probably need to create a new **Parameter Group**
and set it to the database instance).

When adding a monitoring instance for RDS,
specify a unique name to distinguish it from the local MySQL instance.
If you do not specify a name, it will use the client's host name.

It is also recommended to use the ``--create-user`` option,
which will create a dedicated user for the corresponding service.
This is more secure than using a highly privileged account for monitoring.

The following example shows how to enable QAN and MySQL metrics monitoring
on Amazon RDS:

.. code-block:: bash

   # pmm-admin add mysql --host rds-mysql57.vb81uqbc7tbe.us-west-2.rds.amazonaws.com --user rdsuser --password pass --create-user rds-mysql57

.. note:: General system metrics cannot be monitored remotely,
   because ``node_exporter`` requires access to the local file system.
   This means that the ``linux:metrics`` service cannot be used
   to monitor Amazon RDS or any remote database instance.

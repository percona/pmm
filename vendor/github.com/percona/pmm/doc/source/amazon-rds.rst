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

Query analytics requires :ref:`perf-schema` as the query source,
because the slow query log is stored on AWS side,
and QAN agent is not able to read it.
Enable the ``performance_schema`` option under **Parameter Groups** on RDS
(you will probably need to create a new **Parameter Group**
and set it to the database instance).

It also requires the ``statements_digest`` and ``events_statements_history``
to be enabled on the RDS instance.
For more information, see :ref:`perf-schema-settings`.

.. note:: Because of the previous requirements,
   it is not possible to collect query analytics for RDS
   running MySQL version prior to 5.6.
   For MySQL version 5.5 on RDS, see :ref:`cloudwatch`.

When adding a monitoring instance for RDS,
specify a unique name to distinguish it from the local MySQL instance.
If you do not specify a name, it will use the client's host name.

Create the ``pmm`` user with the following privileges
on the RDS instance that you want to monitor::

 GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO 'pmm'@'%' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
 GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@'%';

If you have RDS with MySQL version prior to 5.7,
`REPLICATION CLIENT` privilege is not available there
and has to be excluded from the above statement.

The following example shows how to enable QAN and MySQL metrics monitoring
on Amazon RDS:

.. code-block:: bash

   $ sudo pmm-admin add mysql:metrics --host rds-mysql57.vb81uqbc7tbe.us-west-2.rds.amazonaws.com --user pmm --password pass rds-mysql57
   $ sudo pmm-admin add mysql:queries --host rds-mysql57.vb81uqbc7tbe.us-west-2.rds.amazonaws.com --user pmm --password pass rds-mysql57

.. note:: General system metrics cannot be monitored remotely,
   because ``node_exporter`` requires access to the local file system.
   This means that the ``linux:metrics`` service cannot be used
   to monitor Amazon RDS instances or any remote MySQL instance.

.. _cloudwatch:

Monitoring Amazon RDS OS Metrics
================================

You can use CloudWatch as the data source in Grafana
to monitor OS metrics for Amazon RDS instances.
PMM provides the *Amazon RDS OS Metrics* dashboard for this.

.. image:: images/amazon-rds-os-metrics.png

To set up OS metrics monitoring for Amazon RDS in PMM via CloudWatch:

1. Create an IAM user on the AWS panel for accessing CloudWatch data,
   and attach the managed policy ``CloudWatchReadOnlyAccess`` to it.

#. Create a credentials file on the host running PMM Server
   with the following contents::

    [default]
    aws_access_key_id = <your_access_key_id>
    aws_secret_access_key = <your_secret_access_key>

#. Start the ``pmm-server`` container with an additional ``-v`` flag
   that specifies the location of the file with the IAM user credentials
   and mounts it to :file:`/usr/share/grafana/.aws/credentials`
   in the container. For example:

   .. code-block:: bash

      $ docker run -d \
        -p 80:80 \
        --volumes-from pmm-data \
        -v /path/to/file/with/creds:/usr/share/grafana/.aws/credentials \
        --name pmm-server \
        --restart always \
        percona/pmm-server:latest

The *Amazon RDS OS Metrics* dashboard uses 60 second resolution
and shows the average value for each data point.
An exception is the *CPU Credit Usage* graph,
which has a 5 minute average and interval length.
All data is fetched in real time and not stored anywhere.

This dashboard can be used with any Amazon RDS database engine,
including MySQL, Aurora, etc.

.. note:: Amazon provides one million CloudWatch API requests
   per month at no additional cost.
   Past this, it costs $0.01 per 1,000 requests.
   The pre-defined dashboard performs 15 requests on each refresh
   and an extra two on initial loading.

   For more information, see
   `Amazon CloudWatch Pricing <https://aws.amazon.com/cloudwatch/pricing/>`_.


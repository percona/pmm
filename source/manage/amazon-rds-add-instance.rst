.. _pmm.amazon-rds:

--------------------------------------------------------------------------------
Adding an Amazon RDS MySQL, Aurora MySQL, or Remote Instance
--------------------------------------------------------------------------------


The *PMM Add Instance* is now a preferred method of adding an Amazon RDS
database instance to PMM. This method supports Amazon RDS database instances
that use Amazon Aurora, MySQL, or MariaDB engines, as well as any remote PostgreSQL, ProxySQL, MySQL and MongoDB instances.

Following steps are needed to add an Amazon RDS database instance to PMM:

1. Open the PMM web interface and select the *PMM Add Instance* dashboard.

   .. figure:: ../.res/graphics/png/pmm-add-instance.png

   Choosing the PMM *Add instance* menu entry

#. Select the *Add an AWS RDS MySQL or Aurora MySQL Instance* option in the dashboard.
#. Enter the access key ID and the secret access key of your IAM user.

   .. _figure.pmm.amazon-rds.pmm-server.add-instance.access-key-id:

   .. figure:: ../.res/graphics/png/metrics-monitor.add-instance.png

      Enter the access key ID and the secret access key of your IAM user

#. Click the *Discover* button for PMM to retrieve the available Amazon RDS
   instances.

   .. _figure.pmm.amazon-rds.pmm-server.add-instance.displaying:

   .. figure:: ../.res/graphics/png/metrics-monitor.add-instance.1.png

      PMM displays the available Amazon RDS instances

   For the instance that you would like to monitor, select the
   *Start monitoring* button.

#. You will see a new page with the number of fields. The list is divided into
   the following groups: *Main details*, *RDS database*, *Labels*, and
   *Additional options*. Some already known data, such as already entered
   *AWS access key*, are filled in automatically, and some fields are optional.

   .. _figure.pmm.amazon-rds.pmm-server.add-instance.rds-instances.main-details:

   .. figure:: ../.res/graphics/png/metrics-monitor.add-instance.rds-instances.1.png

      Configuring the selected RDS or Amazon Aurora instance: the
      *Main details* section

   The *Main details* section allows to specify the DNS hostname of your instance,
   service name to use within PMM, the port your service is listening on, the
   database user name and password.

   .. _figure.pmm.amazon-rds.pmm-server.add-instance.rds-instances.labels:

   .. figure:: ../.res/graphics/png/metrics-monitor.add-instance.rds-instances.3.png

      Configuring the selected RDS or Amazon Aurora instance: the
      *Labels* section

   The *Labels* section allows specifying labels for the environment, the AWS
   region and availability zone to be used, the Replication set and Cluster
   names and also it allows to set the list of custom labels in a key:value
   format.

   .. _figure.pmm.amazon-rds.pmm-server.add-instance.rds-instances.additional:

   .. figure:: ../.res/graphics/png/metrics-monitor.add-instance.rds-instances.4.png

      Configuring the selected RDS or Amazon Aurora instance: the
      *Additional options* section for the remote MySQL databse

   The *Additional options* section contains specific flags which allow to tune
   the RDS monitoring. They can allow you to skip connection check, to use TLS
   for the database connection, not to validate the TLS certificate and the
   hostname, as well as to disable basic and/or enhanced metrics collection for
   the RDS instance to reduce costs.

   Also this section contains a database-specific flag, which would allow Query
   Analytics for the selected remote database:

   * when adding some remote MySQL, AWS RDS MySQL or Aurora MySQL instance, you
     will be able to choose using performance schema for the database monitoring
   * when adding a PostgreSQL instance, you will be able to activate using
     ``pg_stat_statements`` extension
   * when adding a MongoDB instance, you will be able to choose using
     QAN MongoDB profiler

#. Finally press the *Add service* button to start monitoring your instance.

.. seealso::

   AWS Documentation: Managing access keys of IAM users
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html

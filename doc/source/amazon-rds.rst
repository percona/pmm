.. _amazon-rds:

================================================================================
Using |pmm| with |amazon-rds|
================================================================================

It is possible to use |pmm| for monitoring |amazon-rds| (just like any remote
|mysql| instance). In this case, the |pmm-client| is not installed on the host
where the database server is deployed. By using the |pmm| web interface, you
connect to the |amazon-rds| DB instance. You only need to provide the |iam| user
access key and |pmm| discovers the |amazon-rds| DB instances available for
monitoring.

.. seealso::

   How do I use the |pmm-add-instance| dashboard to discover |amazon-rds| DB instances?
      pmm.amazon-rds.pmm-add-instance-dashboard.connecting

First of all, ensure that there is minimal latency between |pmm-server| and the
|amazon-rds| instance.

.. note:: If latency is higher than 1 second, you should change the minimum
	  resolution by setting the |term.metrics-resolution| environment
	  variable when :ref:`creating and running the PMM Server container
	  <server-container>`.  For more information, see
	  :ref:`metrics-resolution`.

Network connectivity can become an issue for |prometheus| to scrape
metrics with 1 second resolution.  We strongly suggest that you run
|pmm-server| on |aws.name|.

.. seealso::

   Which ports should be open?
      See :term:`Ports` in glossary
   |amazon-rds| Documentation: Setting Up
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html
   |amazon-rds| Documentation: Getting started
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.html
   |amazon-rds| Documentation: Creating a MySQL DB Instance
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.CreatingConnecting.MySQL.html

.. _pmm.amazon-rds.iam-user.creating:
      
Creating an |iam| user with permission to access |amazon-rds| DB instances
================================================================================

It is recommended that you use an |aws-iam| user account to access |amazon-rds|
DB instances instead of using your |aws| account. This measure improves security
as the permissions of an |iam| user account can be limited so that this account
only grants access to your |amazon-rds| DB instances. On the other
hand, you use your |aws| account to access all |aws| services.

The procedure for creating IAM user accounts is well documented in the
|amazon-rds| documentation. This section only goes through the essential steps
and points out the steps required for using |amazon-rds| with |pmm.name|. 

.. seealso::

   |amazon-rds| Documentation: Creating an IAM user
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM

The first step is to define a policy which will hold all the necessary
permissions. Then, you need to associate this policy with the IAM user or
group. In this section, we will create a new user for this purpose.

.. _pmm.amazon-rds.iam-user.policy:

Creating a policy
--------------------------------------------------------------------------------

A policy defines how |aws| services can be accessed. Once defined it can be
associated with an existing user or group.

To define a new policy use the |aws-iam.name| page at |aws|.

.. figure:: .res/graphics/png/aws.iam.png

   The |aws-iam.name| page at |aws|

1. Select the |gui.policies| option on the navigation panel and click the
   |gui.create-policy| button.
#. On the |gui.create-policy| page, select the |json| tab and replace the
   existing contents with the |json| file provided from :term:`PMM User Permissions for AWS`.
#. Click |gui.review-policy| and set a name to your policy, such as
   |policy-name|. Then, click the |gui.create-policy| button.

.. figure:: .res/graphics/png/aws.iam.create-policy.png

   A new policy is ready to be created.

.. seealso::

   |aws| Documenation: Creating |iam| policies
      https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_create.html

Creating an |iam| user
--------------------------------------------------------------------------------   
   
Policies are attached to existing |iam| users or groups. To create a new |iam|
user, select |gui.users| on the |aws-iam.name| page at |aws|. Then click
|gui.add-user| and complete the following steps:

.. figure:: .res/graphics/png/aws.iam-users.1.png

   Navigate to  |gui.users| on the IAM console

1. On the |gui.add-user| page, set the user name and select the
   |gui.aws-management-console-access| option under
   |gui.select-aws-access-type|. Set a custom password and then proceed to
   permissions by clicking the |gui.permissions| button.
#. On the |gui.set-permissions| page, add the new user to one or more groups if
   necessary. Then, click |gui.review|.
#. On the |gui.add-user| page, click |gui.create-user|.

.. seealso::

   |aws| Documentation: Creating |iam| users
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM
   |aws| Documentation: IAM roles
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html

Attaching a policy to an |iam| user
--------------------------------------------------------------------------------

The last step before you are ready to create an |amazon-rds| DB instance is to
attach the policy with the required permissions to the |iam| user.

First, make sure that the |aws-iam.name| page is open and open
|gui.users|. Then, locate and open the |iam| user that you plan to use with
|amazon-rds| DB instances. Complete the following steps, to apply the policy:

1. On the |gui.permissions| tab, click the |gui.add-permissions| button.
#. On the |gui.add-permissions| page, click |gui.attach-existing-policies-directly|.
#. Using the |gui.filter|, locate the policy with the required permissions (such as |policy-name|).
#. Select a checkbox next to the name of the policy and click |gui.review|.
#. The selected policy appears on the |gui.permissions-summary| page. Click |gui.add-permissions|.

The |policy-name| is now added to your |iam| user.
   
.. figure:: .res/graphics/png/aws.iam.add-permissions.png

   To attach, find the policy on the list and place a check mark to select it
	      
.. seealso::

   Creating an |iam| policy for |pmm|
      :ref:`pmm.amazon-rds.iam-user.policy`

Setting up the |amazon-rds| DB Instance
--------------------------------------------------------------------------------

|qan.name| requires :ref:`perf-schema` as the query source, because
the slow query log is stored on the |amazon-aws| side, and |qan| agent is not able
to read it.  Enable the ``performance_schema`` option under
|gui.parameter-groups| in |amazon-rds|.

.. seealso::

   Performance schema settings
      See :ref:`perf-schema-settings`.
   |aws| Documentation: Parameter groups
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithParamGroups.html

.. note::

   It is not possible to collect query analytics for |amazon-rds|
   running a |mysql| version prior to 5.6.  For |mysql| version 5.5 on
   |amazon-rds|, see :ref:`cloudwatch`.

When adding a monitoring instance for |amazon-rds|,
specify a unique name to distinguish it from the local |mysql| instance.
If you do not specify a name, it will use the client's host name.

Create the ``pmm`` user with the following privileges
on the |amazon-rds| instance that you want to monitor::

 GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO 'pmm'@'%' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
 GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@'%';

If you have |amazon-rds| with a |mysql| version prior to 5.5,
`REPLICATION CLIENT` privilege is not available there
and has to be excluded from the above statement.

.. note::

   General system metrics are monitored by using the |rds-exporter| |prometheus|
   exporter which replaces |node-exporter|. |rds-exporter| gives acces to
   |amazon-cloudwatch| metrics.

   |node-exporter|, used in versions of |pmm| prior to 1.8.0, was not able to
   monitor general system metrics remotely.

The following example shows how to enable |qan| and |mysql| metrics monitoring
on |amazon-rds|:

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.add.mysql-metrics.rds+
   :end-before: #+end-block

.. seealso::

   |aws| Documentation: Connecting to a DB instance (|mysql| engine)
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_ConnectToInstance.html

.. _cloudwatch:

Monitoring |amazon-rds| OS Metrics
================================================================================

|pmm| provides the |amazon-rds-aurora-mysql-metrics| dashboard to monitor
|amazon-rds| instances. The metrics are collected by using the |rds-exporter|
developed and maintained by the |pmm| team. 

.. seealso::

   |rds-exporter| at |github|
      https://github.com/percona/rds_exporter

To set up OS metrics monitoring for |rds| in |pmm| via |cloudwatch|:

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

   .. include:: .res/code/sh.org
      :start-after: +docker.run.iam-user-credential+
      :end-before: #+end-block

The |amazon-rds-aurora-mysql-metrics| dashboard uses the 60 second resolution
and shows the average value for each data point.  An exception is the
|cpu-credit-usage| graph, which has a 5 minute average and interval length.  All
data is fetched in real time and not stored anywhere.

This dashboard can be used with any |amazon-rds| database engine,
including |mysql|, |amazon-aurora|, etc.

.. note:: |amazon| provides one million |amazon-cloudwatch| API requests
   per month at no additional cost.
   Past this, it costs $0.01 per 1,000 requests.
   The pre-defined dashboard performs 15 requests on each refresh
   and an extra two on initial loading.

   For more information, see
   `Amazon CloudWatch Pricing <https://aws.amazon.com/cloudwatch/pricing/>`_.

.. _pmm.amazon-rds.pmm-add-instance-dashboard.connecting:
   
Connecting to an |amazon-rds| DB instance using the |pmm-add-instance| dashboard
================================================================================

.. versionadded:: 1.5.0

The |pmm-add-instance| is now a preferred method to add an |amazon-rds| instance
to |pmm|:

.. figure:: .res/graphics/png/pmm.metrics-monitor.add-instance.png
   
   Enter the access key ID and the secret access key of your |iam| user to view
   |amazon-rds| DB instances.

1. Open the |pmm| web interface and select the |pmm-add-instance| dashboard.
#. Select the |gui.add-rds-aurora-instance| option in the dashboard.
#. Enter the access key ID and the secret access key of your |iam| user.
#. Click the |gui.discover| button for |pmm| to retrieve the available |amazon-rds|
   instances.

.. figure:: .res/graphics/png/pmm.metrics-monitor.add-instance.1.png

   |pmm| displays the available |amazon-rds| instances

For each instance that you would like to monitor, select the |gui.enabled| button
and enter the user name and password. Click |gui.connect|. You can now monitor your
instances in the |amazon-rds-aurora-mysql-metrics|.

.. figure:: .res/graphics/png/pmm.metrics-monitor.add-instance.rds-instances.1.png

   *Enter the DB user name and password to connect to the selected* |rds| or
   |aurora| *instance*.

.. seealso::

   |aws| Documentation: Managing access keys of |iam| users
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html
 
.. |policy-name| replace:: *AmazonRDSforPMMPolicy*

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/option.txt

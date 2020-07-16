.. _pmm.amazon-rds.essential-aws-setting.amazon-rds.db-instance.monitoring:

-----------------------------------------------------------------------------------------------------------------
`Required AWS settings <amazon-rds.html#pmm-amazon-rds-essential-aws-setting-amazon-rds-db-instance-monitoring>`_
-----------------------------------------------------------------------------------------------------------------

It is possible to use |pmm| for monitoring |amazon-rds| (just like any remote
|mysql| instance). In this case, the |pmm-client| is not installed on the host
where the database server is deployed. By using the |pmm| web interface, you
connect to the |amazon-rds| DB instance. You only need to provide the |iam| user
access key (or assign an IAM role) and |pmm| discovers the |amazon-rds| DB
instances available for monitoring.

First of all, ensure that there is the minimal latency between |pmm-server| and the
|amazon-rds| instance.

Network connectivity can become an issue for |prometheus| to scrape
metrics with 1 second resolution.  We strongly suggest that you run
|pmm-server| on |abbr.aws| in the same availability zone as
|amazon-rds| instances.

It is crucial that *enhanced monitoring* be enabled for the |amazon-rds| DB
instances you intend to monitor.

.. _figure.pmm.amazon-rds.amazon-rds.modify-db-instance:

.. figure:: ../.res/graphics/png/amazon-rds.modify-db-instance.2.png

   Set the |gui.enable-enhanced-monitoring| option in the settings of your
   |amazon-rds| DB instance.

.. seealso::

   |amazon-rds| Documentation: 
      - `Modifying an Amazon RDS DB Instance <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html>`_
      - `More information about enhanced monitoring <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.OS.html>`_
      - `Setting Up <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html>`_
      - `Getting started <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.html>`_
      - `Creating a MySQL DB Instance <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.CreatingConnecting.MySQL.html>`_
      - `Availability zones <https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html>`_
      - `What privileges are automatically granted to the master user of an Amazon RDS DB instance? 
	<https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.MasterAccounts.html>`_
   
.. contents::
   :local:

.. _pmm.amazon-rds.permission-access-db-instance.iam-user.creating:
      
`Creating an IAM user with permission to access Amazon RDS DB instances <amazon-rds.html#pmm-amazon-rds-permission-access-db-instance-iam-user-creating>`_
-----------------------------------------------------------------------------------------------------------------------------------------------------------

It is recommended that you use an |iam| user account to access |amazon-rds|
DB instances instead of using your |aws| account. This measure improves security
as the permissions of an |iam| user account can be limited so that this account
only grants access to your |amazon-rds| DB instances. On the other
hand, you use your |aws| account to access all |aws| services.

The procedure for creating |iam| user accounts is well described in the
|amazon-rds| documentation. This section only goes through the essential steps
and points out the steps required for using |amazon-rds| with |percona-monitoring-management|. 

.. seealso::

   |amazon-rds| Documentation: Creating an IAM user
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM

The first step is to define a policy which will hold all the necessary
permissions. Then, you need to associate this policy with the IAM user or
group. In this section, we will create a new user for this purpose.

.. _pmm.amazon-rds.iam-user.policy:

`Creating a policy <amazon-rds.html#pmm-amazon-rds-iam-user-policy>`_
--------------------------------------------------------------------------------

A policy defines how |aws| services can be accessed. Once defined it can be
associated with an existing user or group.

To define a new policy use the |iam| page at |aws|.

.. _figure.pmm.amazon-rds.aws.iam:

.. figure:: ../.res/graphics/png/aws.iam.png

   The |iam| page at |aws|

1. Select the |gui.policies| option on the navigation panel and click the
   |gui.create-policy| button.
#. On the |gui.create-policy| page, select the |json| tab and replace the
   existing contents with the following |json| document.

   .. include:: ../.res/code/aws.iam-user.permission.txt
   
#. Click |gui.review-policy| and set a name to your policy, such as
   |policy-name|. Then, click the |gui.create-policy| button.

.. _figure.pmm.amazon-rds.aws.iam.create-policy:

.. figure:: ../.res/graphics/png/aws.iam.create-policy.png

   A new policy is ready to be created.
   
.. seealso::

   |aws| Documenation: Creating |iam| policies
      https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_create.html

.. _pmm.amazon-rds.iam-user.creating:

`Creating an IAM user <amazon-rds.html#pmm-amazon-rds-iam-user-creating>`_
--------------------------------------------------------------------------------   
   
Policies are attached to existing |iam| users or groups. To create a new |iam|
user, select |gui.users| on the |identity-access-management| page at |aws|. Then click
|gui.add-user| and complete the following steps:

.. _figure.pmm.amazon-rds.aws.iam-users:

.. figure:: ../.res/graphics/png/aws.iam-users.1.png

   Navigate to |gui.users| on the IAM console

1. On the |gui.add-user| page, set the user name and select the
   |gui.programmatic-access| option under
   |gui.select-aws-access-type|. Set a custom password and then proceed to
   permissions by clicking the |gui.permissions| button.
#. On the |gui.set-permissions| page, add the new user to one or more groups if
   necessary. Then, click |gui.review|.
#. On the |gui.add-user| page, click |gui.create-user|.

.. seealso::

   |aws| Documentation: 
      - `Creating IAM users <https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM>`_
      -  `IAM roles <https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html>`_

.. _pmm.amazon-rds.iam-user.access-key.creating:

`Creating an access key for an IAM user <amazon-rds.html#pmm-amazon-rds-iam-user-access-key-creating>`_
--------------------------------------------------------------------------------------------------------

In order to be able to discover an |amazon-rds| DB instance in |pmm|, you either
need to use the access key and secret access key of an existing |iam| user or an
|iam| role. To create an access key for use with |pmm|, open the |iam| console
and click |gui.users| on the navigation pane. Then, select your |iam| user.

To create the access key, open the |gui.security-credentials| tab and click the
|gui.create-access-key| button. The system automatically generates a new access
key ID and a secret access key that you can provide on the |pmm-add-instance|
dashboard to have your |amazon-rds| DB instances discovered.

.. important:: 

   You may use an |iam| role instead of |iam| user provided your |amazon-rds| DB
   instances are associated with the same |aws| account as |pmm|.

In case, the |pmm-server| and |amazon-rds| DB instance were created by using the
same |aws| account, you do not need create the access key ID and secret access
key manually. |pmm| retrieves this information automatically and attempts to
discover your |amazon-rds| DB instances.

.. seealso::

   |aws| Documentation: Managing access keys of |iam| users
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html

.. _pmm.amazon-rds.iam-user.policy.attaching:

`Attaching a policy to an IAM user <amazon-rds.html#pmm-amazon-rds-iam-user-policy-attaching>`_
-----------------------------------------------------------------------------------------------

The last step before you are ready to create an |amazon-rds| DB instance is to
attach the policy with the required permissions to the |iam| user.

First, make sure that the |identity-access-management| page is open and open
|gui.users|. Then, locate and open the |iam| user that you plan to use with
|amazon-rds| DB instances. Complete the following steps, to apply the policy:

1. On the |gui.permissions| tab, click the |gui.add-permissions| button.
#. On the |gui.add-permissions| page, click |gui.attach-existing-policies-directly|.
#. Using the |gui.filter|, locate the policy with the required permissions (such as |policy-name|).
#. Select a checkbox next to the name of the policy and click |gui.review|.
#. The selected policy appears on the |gui.permissions-summary| page. Click |gui.add-permissions|.

The |policy-name| is now added to your |iam| user.
   
.. _figure.pmm.amazon-rds.aws.iam.add-permissions:

.. figure:: ../.res/graphics/png/aws.iam.add-permissions.png

   To attach, find the policy on the list and place a check mark to select it
	      
.. seealso::

   Creating an |iam| policy for |pmm|
      :ref:`pmm.amazon-rds.iam-user.policy`

.. _pmm.amazon-rds.db-instance.setting-up:

`Setting up the Amazon RDS DB Instance <amazon-rds.html#pmm-amazon-rds-db-instance-setting-up>`_
-------------------------------------------------------------------------------------------------

|query-analytics| requires :ref:`perf-schema` as the query source, because the slow
query log is stored on the |abbr.aws| side, and |qan| agent is not able to
read it.  Enable the ``performance_schema`` option under ``Parameter Groups``
in |amazon-rds|.

.. warning:: Enabling Performance Schema on T2 instances is not recommended
   because it can easily run the T2 instance out of memory.

.. seealso::

   More information about the performance schema
      See :ref:`perf-schema`.
   |aws| Documentation: Parameter groups
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithParamGroups.html

When adding a monitoring instance for |amazon-rds|, specify a unique name to
distinguish it from the local |mysql| instance.  If you do not specify a name,
it will use the client's host name.

Create the ``pmm`` user with the following privileges on the |amazon-rds|
instance that you want to monitor::

 GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO 'pmm'@'%' IDENTIFIED BY 'pass' WITH MAX_USER_CONNECTIONS 10;
 GRANT SELECT, UPDATE, DELETE, DROP ON performance_schema.* TO 'pmm'@'%';

If you have |amazon-rds| with a |mysql| version prior to 5.5, `REPLICATION
CLIENT` privilege is not available there and has to be excluded from the above
statement.

.. note::

   General system metrics are monitored by using the |rds-exporter| |prometheus|
   exporter which replaces |node-exporter|. |rds-exporter| gives acces to
   |amazon-cloudwatch| metrics.

   |node-exporter|, used in versions of |pmm| prior to 1.8.0, was not able to
   monitor general system metrics remotely.

.. seealso::

   |aws| Documentation: Connecting to a DB instance (|mysql| engine)
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_ConnectToInstance.html
      
.. |policy-name| replace:: *AmazonRDSforPMMPolicy*

.. include:: ../.res/replace.txt

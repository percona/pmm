.. _pmm.amazon-rds:

================================================================================
Connecting to an |amazon-rds| DB instance
================================================================================

.. versionadded:: 1.5.0

The |pmm-add-instance| is now a preferred method of adding an |amazon-rds| DB
instance to |pmm|. It supports |amazon-rds| DB instances that use
|amazon-aurora|, |mysql|, or |mariadb| engines.

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

   Enter the DB user name and password to connect to the selected* |rds| or
   |aurora| instance.

.. seealso::

   |aws| Documentation: Managing access keys of |iam| users
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html

Essential |aws| settings for monitoring |amazon-rds| DB instances in |pmm|
================================================================================

It is possible to use |pmm| for monitoring |amazon-rds| (just like any remote
|mysql| instance). In this case, the |pmm-client| is not installed on the host
where the database server is deployed. By using the |pmm| web interface, you
connect to the |amazon-rds| DB instance. You only need to provide the |iam| user
access key (or assign an IAM role) and |pmm| discovers the |amazon-rds| DB
instances available for monitoring.

First of all, ensure that there is minimal latency between |pmm-server| and the
|amazon-rds| instance.

Network connectivity can become an issue for |prometheus| to scrape
metrics with 1 second resolution.  We strongly suggest that you run
|pmm-server| on |aws.name| in the same availability zone as |amazon-rds| instances.

.. seealso::

   Which ports should be open?
      See :term:`Ports` in glossary
   |amazon-rds| Documentation: Setting Up
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html
   |amazon-rds| Documentation: Getting started
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.html
   |amazon-rds| Documentation: Creating a MySQL DB Instance
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_GettingStarted.CreatingConnecting.MySQL.html
   |aws| Documentation: Availability zones
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
   |aws| Documentation: What privileges are automatically granted to the master user of an |amazon-rds| DB instance?
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.MasterAccounts.html

.. contents::
   :local:

.. _pmm.amazon-rds.iam-user.creating:
      
Creating an |iam| user with permission to access |amazon-rds| DB instances
--------------------------------------------------------------------------------

It is recommended that you use an |aws-iam| user account to access |amazon-rds|
DB instances instead of using your |aws| account. This measure improves security
as the permissions of an |iam| user account can be limited so that this account
only grants access to your |amazon-rds| DB instances. On the other
hand, you use your |aws| account to access all |aws| services.

The procedure for creating |iam| user accounts is well described in the
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
   existing contents with the following |json| document.

   .. include:: .res/code/js.org
      :start-after: +aws.iam-user.permission+
      :end-before: #+end-block
   
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
   |gui.programmatic-access| option under
   |gui.select-aws-access-type|. Set a custom password and then proceed to
   permissions by clicking the |gui.permissions| button.
#. On the |gui.set-permissions| page, add the new user to one or more groups if
   necessary. Then, click |gui.review|.
#. On the |gui.add-user| page, click |gui.create-user|.

.. seealso::

   |aws| Documentation: Creating |iam| users
      https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM
   |aws| Documentation: |iam| roles
      https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html

Attaching a policy to an |iam| user
--------------------------------------------------------------------------------

The last step before you are ready to create an |amazon-rds| DB instance is to
attach the policy with the required permissions to the |iam| user.

On the |aws-iam.name| page, open |gui.users|. Then, locate and open the |iam|
user that you plan to use with |amazon-rds| DB instances. Complete the following
steps to apply the policy:

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

.. _pmm.amazon-rds.iam-user.access-key.creating:

Creating an access key for an |iam| user
--------------------------------------------------------------------------------

In order to be able to discover an |amazon-rds| DB instance in |pmm|, you either
need to use the access key and secret access key of an existing |iam| user or an
|iam| role. To create an access key for use with |pmm|, open the |iam| console
and click |gui.users| on the navigation pane. Then, select your |iam| user.

To create the access key, open the |gui.security-credentials| tab and
click the |gui.create-access-key| button. The system automatically
generates a new access key ID and a secret access key that you can
provide on the |pmm-add-instance| dashboard to have your |amazon-rds|
DB instances discovered.

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

.. |policy-name| replace:: *AmazonRDSforPMMPolicy*

.. include:: .res/replace/name.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/option.txt

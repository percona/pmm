.. _run-server-ami:

Running PMM Server Using AWS Marketplace
********************************************************************************

You can run an instance of |pmm-server| hosted at AWS Marketplace. This
method replaces the outdated method where you would have to accessing
an AMI (Amazon Machine Image) by using its ID, different for each region.

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.home-page.1.png

   The home page of PMM in AWS Marketplace.

Assuming that you have an AWS (Amazon Web Services) account, locate
*Percona Monitoring and Management Server* in `AWS Marketplace
<https://aws.amazon.com/marketplace/pp/B077J7FYGX>`_.

The |gui.pricing-information| section allows to select your region and choose an
instance type in the table that shows the pricing for the software and
infrastructure hosted in the region you have selected (the recommended
EC2 instance type is preselected for you). Note that actual choice will be done
later, and this table serves the information purposes, to plan costs.

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.home-page.2.png

   As soon as you select your region, you can choose the EC2 instance in it and
   see its price. |pmm| comes for no cost, you may only need to pay for the
   infrastructure provided by |amazon|.

.. note::

   Disk space consumed by |pmm-server| depends on the number of hosts under
   monitoring. Each environment will be unique, however consider modeling your data consumption based on `PMM Demo <https://pmmdemo.percona.com/>`_ web site, which consumes ~230MB/host/day, or ~6.9GB/host at the default 30 day retention period. See `this blog post <https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/>`_ for more details.

* Clicking the |gui.continue-to-subscribe| button will proceed to the terms and
  conditions page.
* Clicking |gui.continue-to-configuration| there will bring a new page to start
  setting up your instance. You will be able to re-check the PMM Server version
  and the region. When done, continue to the launch options by clicking
  |gui.continue-to-launch|.

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.0.png

   Percona Monitoring and Management on AWS Marketplace - launch options.

Select the previously chosen instance type from the *EC2 Instance Type*
drop-down menu. Also chose the launch option. Available launch options in the
*Chose Action* drop-down menu include *Launch from Website* and
*Launch through EC2*. The first one is a quick way to make your instance ready.
For more control, use the Manual Launch through EC2 option.

.. _run-server-aws.pmm-instance.1-click-launch-option.setting-up:

`Setting Up a PMM Instance Using the website GUI <aws.html#run-server-aws-pmm-instance-1-click-launch-option-setting-up>`_
===============================================================================================================================

Choose *Launch from Website* option, your region, and the EC2 instance type on
the launch options page. On the previous screenshot, we use the 
:option:`US East (N. Virginia)` region and the :guilabel:`EC2 Instance Type` named
:option:`t2.medium`. To reduce cost, you need to choose the region closest to
your location.

.. _run-server-aws.pmm-instance.1-click-launch-option.vpc.ec2-instance-type:

`Setting up a VPC and an EC2 Instance Type <aws.html#run-server-aws-pmm-instance-1-click-launch-option-vpc-ec2-instance-type>`_
--------------------------------------------------------------------------------------------------------------------------------

 In this demonstration, we use the VPC (virtual private cloud) named
:option:`vpc-484bb12f`. The exact name of VPC may be different from the example
discussed here.

.. _figure.run-server-ami.aws-marketplace.pmm.launch-on-ec2.1-click-launch.vpc.ec2-instance-type:
  
.. figure:: ../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.1.png

   Select VPC in the VPC Settings section.

Instead of a VPC (virtual private cloud) you may choose the :option:`EC2 Classic
(no VPC)` option and use a public cloud.

Selecting a subnet, you effectively choose an availability zone in the selected
region. We recommend that you choose the availability zone where your RDS is
located.

Note that the cost estimation is automatically updated based on your choice.

.. seealso::

   |aws| Documentation: Availability zones
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
   
.. _run-server-aws.security-group.key-pair:

`Limiting Access to the instance: security group and a key pair <aws.html#run-server-aws-security-group-key-pair>`_
--------------------------------------------------------------------------------------------------------------------

In the |gui.security-group| section, which acts like a firewall, you may use the
preselected option :option:`Create new based on seller settings` to create a
security group with recommended settings. In the :guilabel:`Key Pair` select an
already set up EC2 key pair to limit access to your instance.

.. _figure.run-server-ami.aws-marketplace.pmm.launch-on-ec2.1-click-launch.key-pair.selecting:

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png

   Select an already existing key pair (use the EC2 console to create one if necessary)

.. important::

   It is important that the security group allow communication via the the :ref:`following ports
   <Ports>`: *22*, *80*, and *443*. |pmm| should also be able to access port *3306* on
   the RDS that uses the instance.

.. _figure.run-server-ami.aws-marketplace.pmm-launch-on-ec2.1-click-launch.security-group.selecting:

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.2.png

   Select a security group which manages firewall settings.

.. seealso::

   |amazon| Documentation: Security groups
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html
   |amazon| Documentation: Key pairs
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
   |amazon| Documentation: Importing your own public key to |amazon| EC2
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#how-to-generate-your-own-key-and-import-it-to-aws
      
.. _run-server-aws.setting.applying:

`Applying settings <aws.html#run-server-aws-setting-applying>`_
--------------------------------------------------------------------------------

Scroll up to the top of the page to view your settings. Then, click the
:guilabel:`Launch with 1 click` button to continue and adjust your settings in
the :program:`EC2 console`.

.. _figure.run-server-ami.aws-marketplace.pmm.launch-on-ec2.1-click-launch:

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png
	    
   Your instance settings are summarized in a special area. Click
   the Launch with 1 click button to continue.

.. note:: The :guilabel:`Launch with 1 click` button may alternatively be titled
          as :guilabel:`Accept Software Terms & Launch with 1-Click`.

.. _pmm-aws-instance-setting.ec2-console.adjusting:

`Adjusting instance settings in the EC2 Console <aws.html#pmm-aws-instance-setting-ec2-console-adjusting>`_
------------------------------------------------------------------------------------------------------------

Your clicking the :guilabel:`Launch with 1 click` button, deploys your
instance. To continue setting up your instance, run the :program:`EC2
console`. It is available as a link at the top of the page that opens after you
click the :guilabel:`Launch with 1 click` button.

Your instance appears in the :program:`EC2 console` in a table that lists all
instances available to you. When a new instance is only created, it has no
name. Make sure that you give it a name to distinguish from other instances
managed via the :program:`EC2 console`.

.. _figure.run-server-ami.aws-marketplace.ec2-console.pmm:

.. figure:: ../.res/graphics/png/aws-marketplace.ec2-console.pmm.1.png

   The newly created instance selected.

.. _pmm.server.aws.running-instance:

`Running the instance <aws.html#pmm-server-aws-running-instance>`_
--------------------------------------------------------------------------------

After you add your new instance it will take some time to initialize it. When
the AWS console reports that the instance is now in a running state, you many
continue with configuration of |pmm-server|.

.. note::

   When started the next time after rebooting, your instance may acquire another
   IP address. You may choose to set up an elastic IP to avoid this problem.

   .. seealso::

      |amazon| Documentation: Elastic IP Addresses
         http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html

With your instance selected, open its IP address in a web browser. The IP
address appears in the :guilabel:`IPv4 Public IP` column or as value of the
:guilabel:`Public IP` field at the top of the :guilabel:`Properties` panel.

.. _figure.run-server-ami.aws-marketplace.pmm.ec2.properties:

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.ec2.properties.png

   The public IP address of the instance

To run the instance, copy and paste its public IP address to the location bar of
your browser. In the |pmm.name| welcome page that opens, enter the instance ID.

.. _figure.run-server-ami.installation-wizard.ami.instance-id-verification:

.. figure:: ../.res/graphics/png/installation-wizard.ami.instance-id-verification.png

   Entering the instance ID when installing |pmm-server|

You can copy the instance ID from the :guilabel:`Properties` panel of your
instance, select the :guilabel:`Description` tab back in the :program:`EC2
console`. Click the :guilabel:`Copy` button next to the :guilabel:`Instance
ID` field. This button appears as soon as you hover the cursor of your mouse
over the ID.

.. _figure.run-server-ami.aws-marketplace.pmm.ec2.properties.instance-id:

.. figure:: ../.res/graphics/png/aws-marketplace.pmm.ec2.properties.instance-id.png

   Hover the cursor over the instance ID for the Copy button to appear.

Paste the instance in the :guilabel:`Instance ID` field of the |pmm.name|
welcome page and click |gui.submit|.

|pmm-server| provides user access control, and therefore you will need user
credentials to access it:

.. _figure.run-server-ami.installation-wizard.ami.account-credentials:

.. figure:: ../.res/graphics/png/installation-wizard.ami.account-credentials.png

   Create credentials for your instance.

The default user name is ``admin``, and the default password is ``admin`` also.
You will be proposed to change the default password at login if you didn't it.

The |pmm-server| is now ready and the home page opens.

.. _figure.run-server-ami.pmm-server.home-page:

.. figure:: ../.res/graphics/png/pmm.home-page.png

   |pmm-server| home page

You are creating a username and password that will be used for two purposes:

1. authentication as a user to PMM - this will be the credentials you need in order
   to log in to PMM.
#. authentication between PMM Server and PMM Clients - you will
   re-use these credentials when configuring pmm-client for the first time on a
   server, for example:

   |tip.run-this.root|

   .. code-block:: sh

      $ pmm-admin config --server-insecure-tls --server-address=<IP Address>:443

.. note:: **Accessing the instance by using an SSH client.**

   For instructions about how to access your instances by using an SSH client, see
   `Connecting to Your Linux Instance Using SSH 
   <http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html>`_
	     
   Make sure to replace the user name ``ec2-user`` used in this document with
   ``admin``.

.. seealso::

   How to verify that the |pmm-server| is running properly?
      :ref:`deploy-pmm.server-verifying`

   How to connect a |pmm-client| to the |pmm-server|?
      :ref:`deploy-pmm.client_server.connecting`


.. toctree::
   :name: dockertoc
   :maxdepth: 1

   aws-resize
   aws-upgrade

.. include:: ../.res/replace.txt

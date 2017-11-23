.. _run-server-ami:

==============================================
Running PMM Server Using AWS Marketplace
==============================================

.. - static: https://aws.amazon.com/marketplace/pp/B077J7FYGX
   - recommend Run PMM on AWS in the same Availability Zone (traffic cost + latency)
   - Enable performance_schema option in Parameter Groups for RDS
   - Create separate database user for monitoring in RDS
   - Security group should allow these ports open: 22, 80, 443
   - Use elastic IP to avoid ip mod with reboots
   - PMM should be able to access 3306 on RDS (security group allows)
   - PMM: create user according to https://confluence.percona.com/x/XjkOAQ

You can run an instance of |pmm.name| hosted at AWS Marketplace. This
method replaces the outdated method where you would have to accessing
an AMI (Amazon Machine Image) by using its ID, different for each region.

.. _figure/pmm/deploying/aws-marketplace/home-page.1:

.. figure:: ../../images/aws-marketplace.pmm.home-page.1.png

   *The home page of PMM in AWS Marketplace. Click the Continue button to start
   setting up your instance. You can also preselect your region on this screen.*

Assuming that you have an AWS (Amazon Web Services) account, locate
*Percona Monitoring and Management Server 1.4.1* in `AWS Marketplace
<https://aws.amazon.com/marketplace>`_.

.. note:: **Available versions**

   Currently, you can use this method to run an instance of |pmm| version 1.4.1.

Click the :guilabel:`Continue` button to start setting up your instance. There
are two options available to you. The ``1-Click Launch`` option is a quick way
to make your instance ready. For more control, use the ``Manual Launch`` option.

.. figure:: ../../images/aws-marketplace.pmm.launch-on-ec2.png

   *Percona Monitoring and Management is now available from AWS Marketplace*
	    
Setting up a |pmm| instance using the ``1-Click Launch`` option
================================================================================

With the ``1-Click Launch`` tab selected, make sure that all sections match your
preferences. In this demonstration, we use the :option:`US East (N. Verginia)`
region and the VPC (virtual private cloud) named :option:`vpc-aba20dce`. To
reduce cost, you need to choose the region closest to your location.

On the :guilabel:`1-Click Launch` tab, you select your region in the
:guilabel:`Region` section. Note that your choice of the region is preserved
from
:ref:`the previous screen <figure/pmm/deploying/aws-marketplace/home-page.1>`
where it is available next to the :guilabel:`Continue` button.

Setting up a VPC and an EC2 instance type
--------------------------------------------------------------------------------

The VPC that you select or set up determines which configurations of CPU and RAM
are enabled in the :guilabel:`EC2 Instance Type` section. Having selected the
:option:`vpc-aba20dce` in the :guilabel:`VPC Settings` section, we choose
:option:`m4.large`.

.. figure:: ../../images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.1.png

   *Select VPC in the VPC Settings section and then choose an EC2 instance type
   that suits your planned configuration.*

Note that the cost estimation is automatically updated to correspond with your
choice.

Limiting Access to the instance: security group and a key pair
--------------------------------------------------------------------------------

In the Security group, which acts like a firewall, select a preconfigured
security group in the :guilabel:`Security group` section. In the :guilabel:`Key
Pair` select an already set up EC2 key pair to limit access to your instance

.. figure:: ../../images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.2.png

   *Select a security group which manages firewall settings.*
   
Applying settings
--------------------------------------------------------------------------------

Scroll up to the top of the page to view your settings. Then, click the
:guilabel:`Launch with 1 click` button to continue and adjust your settings in
the :program:`EC2 console`.

.. figure:: ../../images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png
	    
   *Your instance settings are summarized in a special area. Click
   the Launch with 1 click button to continue.*

.. _pmm/ami/instance-setting/ec2-console.adjusting:

Adjusting instance settings in the EC2 Console
--------------------------------------------------------------------------------

Your clicking the :guilabel:`Launch with 1 click` button, deploys your
instance. To continue setting up your instance, run the :program:`EC2
console`. It is available as a link at the top of the page that opens after you
click the :guilabel:`Launch with 1 click` button.

.. figure:: ../../images/aws-marketplace.launch-on-ec2.1-click-launch.4.png

   *Adjust your settings in the EC2 console. To run it, click the EC2 Console
   link in the message at the top of the page.*

Your instance appears in the :program:`EC2 console` in a table that lists all
instances available to you. When a new instance is only created, it has no
name. Make sure that you give it a name to distinguish from other instances
managed via the :program:`EC2 console`.

.. figure:: ../../images/aws-marketplace.ec2-console.pmm.1.png

   *The newly created instance selected.*

Launching the instance
--------------------------------------------------------------------------------

After you add your new instance it will take some time to initialize it. When
the :guilabel:`Instance State` contains :option:`running` for your instance, you
can launch it.

With your instance selected, open its IP address to the in a web browser. The IP
address appears in the :guilabel:`IPv4 Public IP` column or as value of the
:guilabel:`Public IP` field at the top of the :guilabel:`Properties` panel.

.. figure:: ../../images/aws-marketplace.pmm.ec2.properties.png

   *To run the instance, copy and paste its public IP address to the location bar
   of your brower.*

In the |pmm.name| welcome page that opens, enter the instance ID in the
:guilabel:`Instance ID` field.

.. figure:: ../../images/aws-marketplace.pmm.ec2.dialog.instance-id.1.png

   *Enter the instance ID on the welcome page.*

You can copy the instance ID in the :guilabel:`Properties` panel of your
instance, select the :guilabel:`Description` tab back in the :program:`EC2
console`. Click the :guilabel:`Copy` button next to the :guilabel:`Instance
ID` field. This button appears as soon as you hover the cursor of your mouse
over the ID.

.. figure:: ../../images/aws-marketplace.pmm.ec2.properties.instance-id.png

   *Hover the cursor over the instance ID for Copy button to appear.*

Paste the instance in the :guilabel:`Instance ID` field of the |pmm.name|
welcome page and click :guilabel:`Submit`.

The next screen offers to create a user and a password that you will later use
to run your instance. Create a user name, assign a password, and click
:guilabel:`Submit`.

.. figure:: ../../images/aws-marketplace.pmm.ec2.dialog.user-name.png

   *Create credentials for your instance.*

The system authentication window then appears for you to use
your newly created credentials. Enter the user name and password that you have
just created. Your instance is now ready.

.. figure:: ../../images/pmm.home-page.1-4-1b.png

   *Percona Monitoring and Management is now ready*

.. Percona provides public Amazon Machine Images (AMI) with |pmm-server|
.. in all regions where `Amazon Web Services (AWS) <https://aws.amazon.com/>`_ are available.
.. You can launch an instance using the web console for the corresponding image:
.. 
.. .. list-table::
..    :header-rows: 1
.. 
..    * - Region
..      - String
..      - AMI ID
..    * - US East (N. Virginia)
..      - ``us-east-1``
..      - `ami-89e44ff3 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-89e44ff3>`_
..    * - US East (Ohio)
..      - ``us-east-2``
..      - `ami-16321d73 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-16321d73>`_
..    * - US West (N. California)
..      - ``us-west-1``
..      - `ami-a43b04c4 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-a43b04c4>`_
..    * - US West (Oregon)
..      - ``us-west-2``
..      - `ami-9815dee0 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-9815dee0>`_
..    * - Canada (Central)
..      - ``ca-central-1``
..      - `ami-854df5e1 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-854df5e1>`_
..    * - EU (Ireland)
..      - ``eu-west-1``
..      - `ami-c9e040b0 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-c9e040b0>`_
..    * - EU (Frankfurt)
..      - ``eu-central-1``
..      - `ami-921396fd <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-921396fd>`_
..    * - EU (London)
..      - ``eu-west-2``
..      - `ami-781f031c <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-781f031c>`_
..    * - Asia Pacific (Singapore)
..      - ``ap-southeast-1``
..      - `ami-f5b1fc96 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-f5b1fc96>`_
..    * - Asia Pacific (Sydney)
..      - ``ap-southeast-2``
..      - `ami-f72dc395 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-f72dc395>`_
..    * - Asia Pacific (Seoul)
..      - ``ap-northeast-2``
..      - `ami-23ab0f4d <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-23ab0f4d>`_
..    * - Asia Pacific (Tokyo)
..      - ``ap-northeast-1``
..      - `ami-d753ffb1 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-d753ffb1>`_
..    * - Asia Pacific (Mumbai)
..      - ``ap-south-1``
..      - `ami-23b8f54c <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-23b8f54c>`_
..    * - South America (São Paulo)
..      - ``sa-east-1``
..      - `ami-482c5724 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-482c5724>`_
.. 
.. 
.. Running from Command Line
.. =========================
.. 
.. 1. Launch the *PMM Server* instance using the ``run-instances`` command
..    for the corresponding region and image.
..    For example:
.. 
..    .. code-block:: bash
.. 
..       aws ec2 run-instances \
..         --image-id ami-dd5f83a7 \
..         --security-group-ids sg-3b6e5e46 \
..         --instance-type t2.micro \
..         --subnet-id subnet-4765a930 \
..         --region us-east-1 \
..         --key-name SSH-KEYNAME
.. 
..    .. note:: Providing the public SSH key is optional.
..       Specify it if you want SSH access to *PMM Server*.
.. 
.. #. Set a name for the instance using the ``create-tags`` command.
..    For example:
.. 
..    .. code-block:: bash
.. 
..       aws ec2 create-tags  \
..         --resources i-XXXX-INSTANCE-ID-XXXX \
..         --region us-east-1 \
..         --tags Key=Name,Value=OWNER_NAME-pmm
.. 
.. #. Get the IP address for accessing *PMM Server* from console output
..    using the ``get-console-output`` command.
..    For example:
.. 
..    .. code-block:: bash
.. 
..       aws ec2 get-console-output \
..         --instance-id i-XXXX-INSTANCE-ID-XXXX \
..         --region us-east-1 \
..         --output text \
..         | grep cloud-init
.. 




Next Steps
==========

:ref:`Verify that PMM Server is running <deploy-pmm.server.verifying>`
by connecting to the PMM web interface using the IP address
from the console output,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.

.. include:: ../../.resources/name.txt

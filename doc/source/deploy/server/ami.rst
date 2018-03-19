.. _run-server-ami:

==============================================
Running PMM Server Using AWS Marketplace
==============================================

You can run an instance of |pmm-server| hosted at AWS Marketplace. This
method replaces the outdated method where you would have to accessing
an AMI (Amazon Machine Image) by using its ID, different for each region.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.home-page.1.png

   The home page of PMM in AWS Marketplace. Click the Continue button to start
   setting up your instance. You can also preselect your region on this screen.

Assuming that you have an AWS (Amazon Web Services) account, locate
*Percona Monitoring and Management Server* in `AWS Marketplace
<https://aws.amazon.com/marketplace>`_.

In the |gui.pricing-information| section, select your region and choose an
instance type in the table that shows the pricing for the software and
infrastructure hosted in the region you have selected. Note that the recommended
EC2 instance type is preselected for you.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.home-page.2.png

   As soon as you select your region, you can choose the EC2 instance in it and
   see its price. |pmm| comes for no cost, you may only need to pay for the
   infrastructure provided by |amazon|.

Click the |gui.continue-to-subscribe| button to start setting up your instance. There
are two options available to you. The ``1-Click Launch`` option is a quick way
to make your instance ready. For more control, use the ``Manual Launch`` option.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.png

   Percona Monitoring and Management is now available from AWS Marketplace
	    
Setting Up a |pmm| Instance Using the 1-Click Launch Option
================================================================================

With the |gui.1-click-launch| tab selected, make sure that all sections match
your preferences. In this demonstration, we use the :option:`US East
(N. Virginia)` region and the VPC (virtual private cloud) named
:option:`vpc-484bb12f`. To reduce cost, you need to choose the region closest to
your location.

.. note::

   The exact name of VPC may be different from the example discussed here.

On the |gui.1-click-launch| tab, select your region in the |gui.region|
section. By default, the region is the same as the one you chose in the
|gui.pricing-information| section.

Setting up a VPC and an EC2 Instance Type
--------------------------------------------------------------------------------

Depending on your choice of a VPC, some configurations of CPU and RAM may be disabled
in the :guilabel:`EC2 Instance Type` section.

In this demonstration, we select the :option:`vpc-aba20dce` in the
:guilabel:`VPC Settings` section. Then, we choose :option:`m4.large` as the EC2
instance type.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.1.png

   Select VPC in the VPC Settings section and then choose an EC2 instance type
   that suits your planned configuration.

Instead of a VPC (virtual private cloud) you may choose the :option:`EC2 Classic
(no VPC)` option and use a public cloud.

Selecting a subnet, you effectively choose an availability zone in the selected
region. We recommend that you choose the availability zone where your RDS is
located.

Note that the cost estimation is automatically updated based on your choice.

.. seealso::

   |aws| Documentation: Availability zones
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
   

Limiting Access to the instance: security group and a key pair
--------------------------------------------------------------------------------

In the |gui.security-group| section, which acts like a firewall, you may use the
preselected option :option:`Create new based on seller settings` to create a
security group with recommended settings. In the :guilabel:`Key Pair` select an
already set up EC2 key pair to limit access to your instance.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png

   Select an already existing key pair (use the EC2 console to create one if necessary)

.. important::

   It is important that the security group allow communication via the following
   ports: *22*, *80*, and *443*. |pmm| should also be able to access port *3306* on
   the RDS that uses the instance.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.2.png

   Select a security group which manages firewall settings.
   
.. seealso::

   |amazon| Documentation: Security groups
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html
   |amazon| Documentation: Key pairs
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
   |amazon| Documentation: Importing your own public key to |amazon| EC2
      https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#how-to-generate-your-own-key-and-import-it-to-aws
      
Applying settings
--------------------------------------------------------------------------------

Scroll up to the top of the page to view your settings. Then, click the
:guilabel:`Launch with 1 click` button to continue and adjust your settings in
the :program:`EC2 console`.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png
	    
   Your instance settings are summarized in a special area. Click
   the Launch with 1 click button to continue.

.. note:: The :guilabel:`Launch with 1 click` button may alternatively be titled
          as :guilabel:`Accept Software Terms & Launch with 1-Click`.

.. _pmm/ami/instance-setting/ec2-console.adjusting:

Adjusting instance settings in the EC2 Console
--------------------------------------------------------------------------------

Your clicking the :guilabel:`Launch with 1 click` button, deploys your
instance. To continue setting up your instance, run the :program:`EC2
console`. It is available as a link at the top of the page that opens after you
click the :guilabel:`Launch with 1 click` button.

Your instance appears in the :program:`EC2 console` in a table that lists all
instances available to you. When a new instance is only created, it has no
name. Make sure that you give it a name to distinguish from other instances
managed via the :program:`EC2 console`.

.. figure:: ../../.res/graphics/png/aws-marketplace.ec2-console.pmm.1.png

   The newly created instance selected.

.. _pmm.server.ami.running-instance:

Running the instance
--------------------------------------------------------------------------------

After you add your new instance it will take some time to initialize it. When
the :guilabel:`Instance State` contains :option:`running` for your instance, you
can run it.

.. note::

   When started the next time after rebooting, your instance may acquire another
   IP address. You may choose to set up an elastic IP to avoid this problem.

   .. seealso::

      |amazon| Documentation: Elastic IP Addresses
         http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html

With your instance selected, open its IP address in a web browser. The IP
address appears in the :guilabel:`IPv4 Public IP` column or as value of the
:guilabel:`Public IP` field at the top of the :guilabel:`Properties` panel.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.ec2.properties.png

   To run the instance, copy and paste its public IP address to the location bar
   of your brower.

To run the instance, copy and paste its public IP address to the location bar of
your brower. In the |pmm.name| welcome page that opens, enter the instance ID in
the :guilabel:`Instance ID` field.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.ec2.dialog.instance-id.1.png

   Enter the instance ID on the welcome page.

You can copy the instance ID in the :guilabel:`Properties` panel of your
instance, select the :guilabel:`Description` tab back in the :program:`EC2
console`. Click the :guilabel:`Copy` button next to the :guilabel:`Instance
ID` field. This button appears as soon as you hover the cursor of your mouse
over the ID.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.ec2.properties.instance-id.png

   Hover the cursor over the instance ID for the Copy button to appear.

Paste the instance in the :guilabel:`Instance ID` field of the |pmm.name|
welcome page and click |gui.submit|.

The next screen offers to create a user and a password that you will later use
to run your instance. Create a user name, assign a password, and click
|gui.submit|.

.. figure:: ../../.res/graphics/png/aws-marketplace.pmm.ec2.dialog.user-name.png

   Create credentials for your instance.

The system authentication window then appears for you to use
your newly created credentials. Enter the user name and password that you have
just created. Your instance is now ready.

.. figure:: ../../.res/graphics/png/pmm.home-page.png

   Percona Monitoring and Management is now ready

.. note:: **Accessing the instance by using an SSH client.**

   For instructions about how to access your instances by using an SSH client, see
   `Connecting to Your Linux Instance Using SSH <http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html>`_
	     
   Make sure to replace the user name ``ec2-user`` used in this document with
   ``admin``.

Next Steps
==========

:ref:`Verify that PMM Server is running <deploy-pmm.server.verifying>`
by connecting to the PMM web interface using the IP address
from the console output,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.

.. seealso::

   AWS Documentation:

   - `Elastic IP Addresses <http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html>`_.
   - `Amazon EC2 Security Groups for Linux Instances <http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html>`_.
   - `Connecting to Your Linux Instance Using SSH <http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html>`_ (use ``admin`` as the user name)

Running PMM Server Using Amazon Machine Images
==============================================

Percona provides public Amazon Machine Images (AMI) with *PMM Server*
in all regions where Amazon Web Services (AWS) is available.
You can launch an instance using the web console
for the corresponding image:

.. list-table::
   :header-rows: 1

   * - Region
     - String
     - AMI ID
   * - US East (N. Virginia)
     - ``us-east-1``
     - `ami-9809f5e5 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-9809f5e5>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-167c4a73 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-167c4a73>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-b5959fd5 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-b5959fd5>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-beef7bc6 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-beef7bc6>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-0d57d069 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-0d57d069>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-37692a4e <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-37692a4e>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-b10a64de <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-b10a64de>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-54ee0933 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-54ee0933>`_
   * - EU (Paris)
     - ``eu-west-3``
     - `ami-3a56e047 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-3a56e047>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-0273277e <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-0273277e>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-6164a503 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-6164a503>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-5707aa39 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-5707aa39>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-4fda9729 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-4fda9729>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-8b653be4 <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-8b653be4>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-391d5755 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-391d5755>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-06083d63 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-06083d63>`_


Running from Command Line
--------------------------------------------------------------------------------

1. Launch the *PMM Server* instance using the ``run-instances`` command
   for the corresponding region and image.
   For example:

   .. code-block:: bash

      aws ec2 run-instances \
        --image-id ami-9809f5e5 \
        --security-group-ids sg-3b6e5e46 \
        --instance-type t2.micro \
        --subnet-id subnet-4765a930 \
        --region us-east-1 \
        --key-name SSH-KEYNAME

   .. note:: Providing the public SSH key is optional.
      Specify it if you want SSH access to *PMM Server*.

#. Set a name for the instance using the ``create-tags`` command.
   For example:

   .. code-block:: bash

      aws ec2 create-tags  \
        --resources i-XXXX-INSTANCE-ID-XXXX \
        --region us-east-1 \
        --tags Key=Name,Value=OWNER_NAME-pmm

#. Get the IP address for accessing *PMM Server* from console output
   using the ``get-console-output`` command.
   For example:

   .. code-block:: bash

      aws ec2 get-console-output \
        --instance-id i-XXXX-INSTANCE-ID-XXXX \
        --region us-east-1 \
        --output text \
        | grep cloud-init

.. include:: ../../.res/replace/name.txt
.. include:: ../../.res/replace/program.txt
.. include:: ../../.res/replace/option.txt

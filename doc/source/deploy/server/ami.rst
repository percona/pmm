.. _run-server-ami:

==============================================
Running PMM Server Using Amazon Machine Images
==============================================

.. - static: https://aws.amazon.com/marketplace/pp/B077J7FYGX
   - recommend Run PMM on AWS in the same Availability Zone (traffic cost + latency)
   - Enable performance_schema option in Parameter Groups for RDS
   - Create separate database user for monitoring in RDS
   - Security group should allow these ports open: 22, 80, 443
   - Use elastic IP to avoid ip mod with reboots
   - PMM should be able to access 3306 on RDS
   - PMM: create user accoding to https://confluence.percona.com/x/XjkOAQ

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
     - `ami-89e44ff3 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-89e44ff3>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-16321d73 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-16321d73>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-a43b04c4 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-a43b04c4>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-9815dee0 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-9815dee0>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-854df5e1 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-854df5e1>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-c9e040b0 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-c9e040b0>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-921396fd <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-921396fd>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-781f031c <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-781f031c>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-f5b1fc96 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-f5b1fc96>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-f72dc395 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-f72dc395>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-23ab0f4d <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-23ab0f4d>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-d753ffb1 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-d753ffb1>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-23b8f54c <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-23b8f54c>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-482c5724 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-482c5724>`_


Running from Command Line
=========================

1. Launch the *PMM Server* instance using the ``run-instances`` command
   for the corresponding region and image.
   For example:

   .. code-block:: bash

      aws ec2 run-instances \
        --image-id ami-dd5f83a7 \
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

Next Steps
==========

:ref:`Verify that PMM Server is running <deploy-pmm.server.verifying>`
by connecting to the PMM web interface using the IP address
from the console output,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.


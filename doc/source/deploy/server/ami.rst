.. _run-server-ami:

==============================================
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
     - `ami-cb569ab1 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-cb569ab1>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-53684436 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-53684436>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-c6794ba6 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-c6794ba6>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-8da96ff5 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-8da96ff5>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-77fd4513 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-77fd4513>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-d44093ad <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-d44093ad>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-12cd727d <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-12cd727d>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-77859713 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-77859713>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-ae86fecd <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-ae86fecd>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-8428cae6 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-8428cae6>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-07f05569 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-07f05569>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-7896491e <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-7896491e>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-4e175421 <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-4e175421>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-c26c12ae <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-c26c12ae>`_

Running from Command Line
=========================

1. Launch the *PMM Server* instance using the ``run-instances`` command
   for the corresponding region and image.
   For example:

   .. code-block:: bash

      aws ec2 run-instances \
        --image-id ami-cb569ab1 \
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


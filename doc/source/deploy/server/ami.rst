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
     - AMI ID
   * - US East (N. Virginia)
     - `ami-78880e6e <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-78880e6e>`_
   * - US East (Ohio)
     - `ami-6ccaee09 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-6ccaee09>`_
   * - US West (N. California)
     - `ami-ba451fda <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-ba451fda>`_
   * - US West (Oregon)
     - `ami-af3daacf <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-af3daacf>`_
   * - Canada (Central)
     - `ami-9400bdf0 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-9400bdf0>`_
   * - EU (Ireland)
     - `ami-a36950c5 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-a36950c5>`_
   * - EU (Frankfurt)
     - `ami-ee2dfd81 <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-ee2dfd81>`_
   * - EU (London)
     - `ami-681e0a0c <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-681e0a0c>`_
   * - Asia Pacific (Singapore)
     - `ami-2f87384c <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-2f87384c>`_
   * - Asia Pacific (Sydney)
     - `ami-f6aca395 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-f6aca395>`_
   * - Asia Pacific (Seoul)
     - `ami-8e9745e0 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-8e9745e0>`_
   * - Asia Pacific (Tokyo)
     - `ami-36dff851 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-36dff851>`_
   * - Asia Pacific (Mumbai)
     - `ami-1541327a <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-1541327a>`_
   * - South America (São Paulo)
     - `ami-f8a3c094 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-f8a3c094>`_

Running from Command Line
=========================

1. Launch the *PMM Server* instance using the ``run-instances`` command
   for the corresponding region and image.
   For example:

   .. code-block:: bash

      aws ec2 run-instances \
        --image-id ami-9a0acb8c \
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

:ref:`Verify that PMM Server is running <verify-server>`
by connecting to the PMM web interface using the IP address
from the console output,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.


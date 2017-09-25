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
     - `ami-c2e0efd4 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-1a668f60>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-bdb091d8 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-6b7b560e>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-ededc38d <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-ffc0f19f>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-8e9589f7 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-b28497d6>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-d050efb4 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-f678c192>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-62e00d1b <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-681fd611>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-50e84a3f <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-8bae1ee4>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-078b9d63 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-078b9d63>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-c762f5a4 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-f2f68691>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-9ab9a4f9 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-ccc829ae>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-180fd176 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-f8b96396>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-71110b16 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-0f844e69>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-9fa7def0 <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-b20849dd>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-5f1c6833 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-196c1075>`_

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

:ref:`Verify that PMM Server is running <deploy-pmm.server.verifying>`
by connecting to the PMM web interface using the IP address
from the console output,
then :ref:`install PMM Client <install-client>`
on all database hosts that you want to monitor.


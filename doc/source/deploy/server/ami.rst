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
     - `ami-058f7d7f <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-058f7d7f>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-fd3f1298 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-fd3f1298>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-09211169 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-09211169>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-0efb0376 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-0efb0376>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-f166df95 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-f166df95>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-c79e54be <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-c79e54be>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-e67ac989 <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-e67ac989>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-16435172 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-16435172>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-ff94e69c <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-ff94e69c>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-1f9d7d7d <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-1f9d7d7d>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-09c51f67 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-09c51f67>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-e538ec83 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-e538ec83>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-6a85c405 <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-6a85c405>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-38f18d54 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-38f18d54>`_

Running from Command Line
=========================

1. Launch the *PMM Server* instance using the ``run-instances`` command
   for the corresponding region and image.
   For example:

   .. code-block:: bash

      aws ec2 run-instances \
        --image-id ami-058f7d7f \
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


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
     - `ami-dd5f83a7 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-dd5f83a7>`_
   * - US East (Ohio)
     - ``us-east-2``
     - `ami-64f5d901 <https://console.aws.amazon.com/ec2/v2/home?region=us-east-2#Images:visibility=public-images;imageId=ami-64f5d901>`_
   * - US West (N. California)
     - ``us-west-1``
     - `ami-731c2113 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-1#Images:visibility=public-images;imageId=ami-731c2113>`_
   * - US West (Oregon)
     - ``us-west-2``
     - `ami-ec63a194 <https://console.aws.amazon.com/ec2/v2/home?region=us-west-2#Images:visibility=public-images;imageId=ami-ec63a194>`_
   * - Canada (Central)
     - ``ca-central-1``
     - `ami-e5a91181 <https://console.aws.amazon.com/ec2/v2/home?region=ca-central-1#Images:visibility=public-images;imageId=ami-e5a91181>`_
   * - EU (Ireland)
     - ``eu-west-1``
     - `ami-802af3f9 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-802af3f9>`_
   * - EU (Frankfurt)
     - ``eu-central-1``
     - `ami-6905bc06 <https://console.aws.amazon.com/ec2/v2/home?region=eu-central-1#Images:visibility=public-images;imageId=ami-6905bc06>`_
   * - EU (London)
     - ``eu-west-2``
     - `ami-3c293458 <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-2#Images:visibility=public-images;imageId=ami-3c293458>`_
   * - Asia Pacific (Singapore)
     - ``ap-southeast-1``
     - `ami-60286f03 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-1#Images:visibility=public-images;imageId=ami-60286f03>`_
   * - Asia Pacific (Sydney)
     - ``ap-southeast-2``
     - `ami-939478f1 <https://console.aws.amazon.com/ec2/v2/home?region=ap-southeast-2#Images:visibility=public-images;imageId=ami-939478f1>`_
   * - Asia Pacific (Seoul)
     - ``ap-northeast-2``
     - `ami-9771d4f9 <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#Images:visibility=public-images;imageId=ami-9771d4f9>`_
   * - Asia Pacific (Tokyo)
     - ``ap-northeast-1``
     - `ami-1b96337d <https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#Images:visibility=public-images;imageId=ami-1b96337d>`_
   * - Asia Pacific (Mumbai)
     - ``ap-south-1``
     - `ami-fe2d6f91 <https://console.aws.amazon.com/ec2/v2/home?region=ap-south-1#Images:visibility=public-images;imageId=ami-fe2d6f91>`_
   * - South America (São Paulo)
     - ``sa-east-1``
     - `ami-994f36f5 <https://console.aws.amazon.com/ec2/v2/home?region=sa-east-1#Images:visibility=public-images;imageId=ami-994f36f5>`_

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


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
     - `ami-ebdf17fd <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=ami-ebdf17fd>`_
   * - EU (Ireland)
     - `ami-394e6c5f <https://console.aws.amazon.com/ec2/v2/home?region=eu-west-1#Images:visibility=public-images;imageId=ami-394e6c5f>`_

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


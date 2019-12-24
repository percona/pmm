.. _aws.ebs-volume.resizing:

Resizing the EBS Volume
--------------------------------------------------------------------------------

Your AWS instance comes with a predefined size which can become a limitation. To
make more disk space available to your instance, you need to increase the size
of the EBS volume as needed and then your instance will reconfigure itself to
use the new size.

The procedure of resizing EBS volumes is described in the |amazon|
documentation: `Modifying the Size, IOPS, or Type of an EBS Volume on Linux 
<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-modify-volume.html>`_.

After the EBS volume is updated, |pmm-server| instance will autodetect changes
in approximately 5 minutes or less and will reconfigure itself for the updated
conditions.

.. include:: ../.res/replace.txt

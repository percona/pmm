.. _upgrade-pmm-server-aws:

Upgrading PMM Server on AWS
================================================================================

.. _upgrade-ec2-instance-class:

`Upgrading EC2 instance class <ami.html#upgrade-ec2-instance-class>`_
--------------------------------------------------------------------------------

Upgrading to a larger EC2 instance class is supported by PMM provided you follow
the instructions from the `AWS manual <https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-resize.html>`_.
The PMM AMI image uses a distinct EBS volume for the PMM data volume which
permits independent resize of the EC2 instance without impacting the EBS volume.

.. _expand-pmm-data-volume:

`Expanding the PMM Data EBS Volume <ami.html#expand-pmm-data-volume>`_
--------------------------------------------------------------------------------

The PMM data volume is mounted as an XFS formatted volume on top of an LVM
volume. There are two ways to increase this volume size:

1. Add a new disk via EC2 console or API, and expand the LVM volume to include
   the new disk volume.
2. Expand existing EBS volume and grow the LVM volume.

Expand existing EBS volume
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
To expand the existing EBS volume in order to increase capacity, the following
steps should be followed.

1. Expand the disk from AWS Console/CLI to the desired capacity.
2. Login to the PMM EC2 instance and verify that the disk capacity has
   increased. For example, if you have expanded disk from 16G to 32G, ``dmesg``
   output should look like below::

     [  535.994494] xvdb: detected capacity change from 17179869184 to 34359738368

3. You can check information about volume groups and logical volumes with the
   ``vgs`` and ``lvs`` commands::

    [root@ip-10-1-2-70 ~]# vgs
     VG     #PV #LV #SN Attr   VSize  VFree
     DataVG   1   2   0 wz--n- <16.00g    0

    [root@ip-10-1-2-70 ~]# lvs
     LV       VG     Attr       LSize   Pool Origin Data%  Meta% Move Log Cpy%Sync Convert
     DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.74
     ThinPool DataVG twi-aotz--  15.96g 1.39  1.29

4. Now we can use the ``lsblk`` command to see that our disk size has been
   identified by the kernel correctly, but LVM2 is not yet aware of the new size.
   We can use ``pvresize`` to make sure the PV device reflects the new size.
   Once ``pvresize`` is executed, we can see that the VG has the new free space
   available.

   .. code-block:: bash

      [root@ip-10-1-2-70 ~]# lsblk | grep xvdb
       xvdb                      202:16 0 32G 0 disk

      [root@ip-10-1-2-70 ~]# pvscan
       PV /dev/xvdb   VG DataVG    lvm2 [<16.00 GiB / 0    free]
       Total: 1 [<16.00 GiB] / in use: 1 [<16.00 GiB] / in no VG: 0 [0   ]

      [root@ip-10-1-2-70 ~]# pvresize /dev/xvdb
       Physical volume "/dev/xvdb" changed
       1 physical volume(s) resized / 0 physical volume(s) not resized

      [root@ip-10-1-2-70 ~]# pvs
       PV         VG     Fmt  Attr PSize   PFree
       /dev/xvdb  DataVG lvm2 a--  <32.00g 16.00g

5. We then extend our logical volume. Since the PMM image uses thin
   provisioning, we need to extend both the pool and the volume::

      [root@ip-10-1-2-70 ~]# lvs
       LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
       DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
       ThinPool DataVG twi-aotz--  15.96g                 1.42   1.32

      [root@ip-10-1-2-70 ~]# lvextend /dev/mapper/DataVG-ThinPool -l 100%VG
       Size of logical volume DataVG/ThinPool_tdata changed from 16.00 GiB (4096 extents) to 31.96 GiB (8183 extents).
       Logical volume DataVG/ThinPool_tdata successfully resized.

      [root@ip-10-1-2-70 ~]# lvs
       LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
       DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
       ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71

6. Once the pool and volumes have been extended, we need to now extend the thin
   volume to consume the newly available space. In this example we've grown
   available space to almost 32GB, and already consumed 12GB, so we're extending
   an additional 19GB:

   .. code-block:: bash

      [root@ip-10-1-2-70 ~]# lvs
       LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
       DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
       ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71

      [root@ip-10-1-2-70 ~]# lvextend /dev/mapper/DataVG-DataLV -L +19G
       Size of logical volume DataVG/DataLV changed from <12.80 GiB (3276 extents) to <31.80 GiB (8140 extents).
       Logical volume DataVG/DataLV successfully resized.

      [root@ip-10-1-2-70 ~]# lvs
       LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
       DataLV   DataVG Vwi-aotz-- <31.80g ThinPool        0.71
       ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71

7. We then expand the XFS filesystem to reflect the new size using
   ``xfs_growfs``, and confirm the filesystem is accurate using the ``df``
   command.

   .. code-block:: bash

      [root@ip-10-1-2-70 ~]# df -h /srv
      Filesystem                  Size Used Avail Use% Mounted on
      /dev/mapper/DataVG-DataLV    13G 249M   13G   2% /srv

      [root@ip-10-1-2-70 ~]# xfs_growfs /srv
      meta-data=/dev/mapper/DataVG-DataLV isize=512    agcount=103, agsize=32752 blks
               =                          sectsz=512   attr=2, projid32bit=1
               =                          crc=1        finobt=0 spinodes=0
      data     =                          bsize=4096   blocks=3354624, imaxpct=25
               =                          sunit=16     swidth=16 blks
      naming   =version 2                 bsize=4096   ascii-ci=0 ftype=1
      log      =internal                  bsize=4096   blocks=768, version=2
               =                          sectsz=512   sunit=16 blks, lazy-count=1
      realtime =none                      extsz=4096   blocks=0, rtextents=0
      data blocks changed from 3354624 to 8335360

      [root@ip-10-1-2-70 ~]# df -h /srv
      Filesystem                 Size Used Avail Use% Mounted on
      /dev/mapper/DataVG-DataLV   32G 254M   32G   1% /srv



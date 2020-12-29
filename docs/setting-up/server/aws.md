# AWS Marketplace

You can run an instance of PMM Server hosted at AWS Marketplace.

Assuming that you have an AWS (Amazon Web Services) account, locate *Percona Monitoring and Management Server* in [AWS Marketplace](https://aws.amazon.com/marketplace) (or use [this link]({{ extra.server_aws }})).

![](../../_images/PMM_AWS_Marketplace.png)

Selecting a region and instance type in the *Pricing Information* section will give you an estimate of the costs involved. This is only an indication of costs. You will choose regions and instance types in later steps.

Percona Monitoring and Management Server is provided at no cost, but you may need to pay for infrastructure costs.

!!! note

    Disk space consumed by PMM Server depends on the number of hosts being monitored. Although each environment will be unique, you can consider the data consumption figures for the [PMM Demo](https://pmmdemo.percona.com/) web site which
    consumes approximately 230MB/host/day, or ~6.9GB/host at the default 30 day retention period.

    For more information, see our blog post [How much disk space should I allocate for Percona Monitoring and Management?](https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/).

1. Click *Continue to Subscribe*.

2. **Subscribe to this software**: Check the terms and conditions and click *Continue to Configuration*.

3. **Configure this software**:

    1. Select a value for *Software Version*. (The latest is {{ release }})
    2. Select a region. (You can change this in the next step.)
    3. Click *Continue to Launch*.

4. **Launch this software**:

    1. **Choose Action**: Select a launch option. *Launch from Website* is a quick way to make your instance ready. For more control, choose *Launch through EC2*.

    2. **EC2 Instance Type**: Select an instance type.

    3. **VPC Settings**: Choose or create a VPC (virtual private cloud).

    4. **Subnet Settings**: Choose or create a subnet.

    5. **Security Group Settings**: Choose a security group or click *Create New Based On Seller Settings

    6. **Key Pair Settings**: Choose or create a key pair.

    7. Click *Launch*.

## Limiting Access to the instance: security group and a key pair

In the *Security Group* section, which acts like a firewall, you may use the preselected option `Create new based on seller settings` to create a security group with recommended settings. In the *Key Pair* select an already set up EC2 key pair to limit access to your instance.

![image](../../_images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png)

!!! note

    It is important that the security group allow communication via the the following ports: *22*, *80*, and *443*. PMM should also be able to access port *3306* on the RDS that uses the instance.

![image](../../_images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.2.png)

## Applying settings

Scroll up to the top of the page to view your settings. Then, click the *Launch with 1 click* button to continue and adjust your settings in the EC2 console.

Your instance settings are summarized in a special area. Click the Launch with 1 click button to continue.

![image](../../_images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png)

!!! note

    The *Launch with 1 click* button may alternatively be titled as *Accept Software Terms & Launch with 1-Click*.

## Adjusting instance settings in the EC2 Console

Your clicking the *Launch with 1 click* button, deploys your instance. To continue setting up your instance, run the EC2 console. It is available as a link at the top of the page that opens after you click the *Launch with 1 click* button.

Your instance appears in the EC2 console in a table that lists all instances available to you. When a new instance is only created, it has no name. Make sure that you give it a name to distinguish from other instances managed via the EC2 console.

![image](../../_images/aws-marketplace.ec2-console.pmm.1.png)

## Running the instance

After you add your new instance it will take some time to initialize it. When the AWS console reports that the instance is now in a running state, you many continue with configuration of PMM Server.

!!! note

    When started the next time after rebooting, your instance may acquire another IP address. You may choose to set up an elastic IP to avoid this problem.

With your instance selected, open its IP address in a web browser. The IP address appears in the *IPv4 Public IP* column or as value of the *Public IP* field at the top of the *Properties* panel.

![image](../../_images/aws-marketplace.pmm.ec2.properties.png)

To run the instance, copy and paste its public IP address to the location bar of your browser. In the *Percona Monitoring and Management* welcome page that opens, enter the instance ID.

![image](../../_images/installation-wizard.ami.instance-id-verification.png)

You can copy the instance ID from the *Properties* panel of your instance, select the *Description* tab back in the EC2 console. Click the *Copy* button next to the *Instance ID* field. This button appears as soon as you hover the cursor of your mouse over the ID.

Hover the cursor over the instance ID for the Copy button to appear.

![image](../../_images/aws-marketplace.pmm.ec2.properties.instance-id.png)

Paste the instance in the *Instance ID* field of the *Percona Monitoring and Management* welcome page and click *Submit*.

PMM Server provides user access control, and therefore you will need user credentials to access it:

![image](../../_images/PMM_Login.jpg)

- Default user name: ``admin``
- Default password: ``admin``

You will be prompted to change the default password every time you log in.

The PMM Server is now ready and the home page opens.

![image](../../_images/PMM_Home_Dashboard.jpg)

You are creating a username and password that will be used for two purposes:

1. authentication as a user to PMM - this will be the credentials you need in order to log in to PMM.

2. authentication between PMM Server and PMM Clients - you will re-use these credentials when configuring pmm-client for the first time on a server, for example:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@<IP Address>:443
    ```

    !!! note "Accessing the instance by using an SSH client"

        For instructions about how to access your instances by using an SSH client, see [Connecting to Your Linux Instance Using SSH](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html)

    Make sure to replace the user name `ec2-user` used in this document with `admin`.

## Resizing the EBS Volume

Your AWS instance comes with a predefined size which can become a limitation. To make more disk space available to your instance, you need to increase the size of the EBS volume as needed and then your instance will reconfigure itself to use the new size.

The procedure of resizing EBS volumes is described in the Amazon documentation: [Modifying the Size, IOPS, or Type of an EBS Volume on Linux](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-modify-volume.html).

After the EBS volume is updated, PMM Server instance will auto-detect changes in approximately 5 minutes or less and will reconfigure itself for the updated conditions.

## Upgrading PMM Server on AWS

### Upgrading EC2 instance class

Upgrading to a larger EC2 instance class is supported by PMM provided you follow the instructions from the [AWS manual](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-resize.html). The PMM AMI image uses a distinct EBS volume for the PMM data volume which permits independent resize of the EC2 instance without impacting the EBS volume.

### Expanding the PMM Data EBS Volume

The PMM data volume is mounted as an XFS formatted volume on top of an LVM volume. There are two ways to increase this volume size:

1. Add a new disk via EC2 console or API, and expand the LVM volume to include the new disk volume.

2. Expand existing EBS volume and grow the LVM volume.

### Expand existing EBS volume

To expand the existing EBS volume in order to increase capacity, the following steps should be followed.

1. Expand the disk from AWS Console/CLI to the desired capacity.

2. Login to the PMM EC2 instance and verify that the disk capacity has increased. For example, if you have expanded disk from 16G to 32G, `dmesg` output should look like below:

    ```
    [  535.994494] xvdb: detected capacity change from 17179869184 to 34359738368
    ```

3. You can check information about volume groups and logical volumes with the `vgs` and `lvs` commands:

    ```sh
    vgs
    ```

    ```
    VG     #PV #LV #SN Attr   VSize  VFree
    DataVG   1   2   0 wz--n- <16.00g    0
    ```

    ```sh
    lvs
    ```

    ```
    LV       VG     Attr       LSize   Pool Origin Data%  Meta% Move Log Cpy%Sync Convert
    DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.74
    ThinPool DataVG twi-aotz--  15.96g 1.39  1.29
    ```

4. Now we can use the `lsblk` command to see that our disk size has been identified by the kernel correctly, but LVM2 is not yet aware of the new size. We can use `pvresize` to make sure the PV device reflects the new size. Once `pvresize` is executed, we can see that the VG has the new free space available.

    ```sh
    lsblk | grep xvdb
    ```

    ```
    xvdb                      202:16 0 32G 0 disk
    ```

    ```sh
    pvscan
    ```

    ```
    PV /dev/xvdb   VG DataVG    lvm2 [<16.00 GiB / 0    free]
    Total: 1 [<16.00 GiB] / in use: 1 [<16.00 GiB] / in no VG: 0 [0   ]
    ```

    ```sh
    pvresize /dev/xvdb
    ```

    ```
    Physical volume "/dev/xvdb" changed
    1 physical volume(s) resized / 0 physical volume(s) not resized
    ```

    ```sh
    pvs
    ```

    ```
    PV         VG     Fmt  Attr PSize   PFree
    /dev/xvdb  DataVG lvm2 a--  <32.00g 16.00g
    ```

5. We then extend our logical volume. Since the PMM image uses thin provisioning, we need to extend both the pool and the volume:

    ```sh
    lvs
    ```

    ```
    LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
    DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
    ThinPool DataVG twi-aotz--  15.96g                 1.42   1.32
    ```

    ```sh
    lvextend /dev/mapper/DataVG-ThinPool -l 100%VG
    ```

    ```
    Size of logical volume DataVG/ThinPool_tdata changed from 16.00 GiB (4096 extents) to 31.96 GiB (8183 extents).
    Logical volume DataVG/ThinPool_tdata successfully resized.
    ```

    ```sh
    lvs
    ```

    ```
    LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
    DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
    ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71
    ```

6. Once the pool and volumes have been extended, we need to now extend the thin volume to consume the newly available space. In this example we’ve grown available space to almost 32GB, and already consumed 12GB, so we’re extending an additional 19GB:

    ```sh
    lvs
    ```

    ```
    LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
    DataLV   DataVG Vwi-aotz-- <12.80g ThinPool        1.77
    ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71
    ```

    ```sh
    lvextend /dev/mapper/DataVG-DataLV -L +19G
    ```

    ```
    Size of logical volume DataVG/DataLV changed from <12.80 GiB (3276 extents) to <31.80 GiB (8140 extents).
    Logical volume DataVG/DataLV successfully resized.
    ```

    ```sh
    lvs
    ```

    ```
    LV       VG     Attr       LSize   Pool    Origin Data%  Meta% Move Log Cpy%Sync Convert
    DataLV   DataVG Vwi-aotz-- <31.80g ThinPool        0.71
    ThinPool DataVG twi-aotz--  31.96g                 0.71   1.71
    ```

7. We then expand the XFS filesystem to reflect the new size using `xfs_growfs`, and confirm the filesystem is accurate using the `df` command.

    ```sh
    df -h /srv
    ```

    ```
    Filesystem                  Size Used Avail Use% Mounted on
    /dev/mapper/DataVG-DataLV    13G 249M   13G   2% /srv
    ```

    ```sh
    xfs_growfs /srv
    ```

    ```
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
    ```

    ```sh
    df -h /srv
    ```

    ```
    Filesystem                 Size Used Avail Use% Mounted on
    /dev/mapper/DataVG-DataLV   32G 254M   32G   1% /srv
    ```

!!! seealso "See also"

    * [Amazon AWS Documentation: Availability zones](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html)
    * [Amazon AWS Documentation: Security groups](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html)
    * [Amazon AWS Documentation: Key pairs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html)
    * [Amazon AWS Documentation: Importing your own public key to Amazon EC2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#how-to-generate-your-own-key-and-import-it-to-aws)
    * [Amazon AWS Documentation: Elastic IP Addresses](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/elastic-ip-addresses-eip.html)

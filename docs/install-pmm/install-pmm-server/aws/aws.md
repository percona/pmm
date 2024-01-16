# Install PMM server on AWS Marketplace

You can run an instance of PMM Server hosted at AWS Marketplace.

Assuming that you have an AWS (Amazon Web Services) account, locate *Percona Monitoring and Management Server* in [AWS Marketplace](https://aws.amazon.com/marketplace) or use [this link](https://aws.amazon.com/marketplace/pp/B077J7FYGX).

![!](../../../_images/PMM_AWS_Marketplace.png)

Selecting a region and instance type in the *Pricing Information* section will give you an estimate of the costs involved. This is only an indication of costs. You will choose regions and instance types in later steps.

Percona Monitoring and Management Server is provided at no cost, but you may need to pay for infrastructure costs.

!!! note alert alert-primary ""
    Disk space consumed by PMM Server depends on the number of hosts being monitored. Although each environment will be unique, you can consider the data consumption figures for the [PMM Demo](https://pmmdemo.percona.com/) web site which consumes approximately 230 MB per host per day, or approximately 6.9 GB per host at the default 30 day retention period.

    For more information, see our blog post [How much disk space should I allocate for Percona Monitoring and Management?](https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/).

To install PMM server on AWS:
{.power-number}

1. Click **Continue to Subscribe**.

2. **Subscribe to this software**: Check the terms and conditions and click *Continue to Configuration*.

3. **Configure this software**:

    1. Select a value for **Software Version**. (The latest is {{release}}.)
    2. Select a region. (You can change this in the next step.)
    3. Click **Continue to Launch**.

4. **Launch this software**:

    1. **Choose Action**: Select a launch option. **Launch from Website** is a quick way to make your instance ready. For more control, choose *Launch through EC2*.

    2. **EC2 Instance Type**: Select an instance type.

    3. **VPC Settings**: Choose or create a VPC (virtual private cloud).

    4. **Subnet Settings**: Choose or create a subnet.

    5. **Security Group Settings**: Choose a security group or click *Create New Based On Seller Settings

    6. **Key Pair Settings**: Choose or create a key pair.

    7. Click **Launch**.



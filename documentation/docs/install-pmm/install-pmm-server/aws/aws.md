# Install PMM Server from AWS Marketplace

To install Percona Monitoring and Management (PMM) Server from AWS Marketplace:
{.power-number}

1. Go to [AWS Marketplace](https://aws.amazon.com/marketplace) and search for **Percona Monitoring and Management Server** or [access the PMM Server listing](https://aws.amazon.com/marketplace/pp/prodview-uww55ejutsnom?sr=0-1&ref_=beagle&applicationId=AWSMPContessa) directly.

2. Click **Continue to Subscribe** on the PMM Server listing page, review the terms and conditions, then click **Continue to Configuration**.

3. Select the latest version (recommended), choose the AWS region where you want to deploy PMM, then click **Continue to Launch**.
4. Choose **Launch from Website** to configure and launch directly from the AWS Marketplace or **Launch through EC2** if you prefer launching via the EC2 Management Console for more customization.
5. In the **EC2 Instance Type** field, select an appropriate instance type based on your monitoring needs and anticipated load. 
6. In the **VPC Settings** field, choose an existing VPC or create a new one to host your PMM Server.
7. In the **Subnet Settings** field, select an existing subnet or create a new one within your VPC.
8. In the **Security Group Settings** field, choose an existing security group or create a new one based on the default settings provided by the seller.
9. In the **Key Pair Settings** field, select an existing key pair for SSH access, or create a new one if necessary.
10. Click **Launch** to deploy the PMM Server.
11. Once the instance is launched, it will appear in the EC2 console.

Make sure to assign a meaningful name to the instance to help distinguish it from others in your environment.

## Security consideration

Ensure that your security group allows inbound traffic on ports **22** (SSH) and **443** (HTTPS).

## Service costs

While PMM Server itself is provided at no cost, be aware that you will incur AWS infrastructure costs based on the EC2 instance type, storage, and data transfer.

## Disk space consumption

The disk space required by PMM Server depends on the number of monitored hosts and the retention period for the data.

As a reference, the [PMM2 Demo](https://pmmdemo.percona.com/) site consumes approximately 230 MB per host per day, which totals around 6.9 GB per host over a 30-day retention period.
Tip: You can estimate your disk space needs based on the number of hosts and the desired retention period.

For more information, see our blog post [How much disk space should I allocate for Percona Monitoring and Management](https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/).

# Install PMM Server from AWS Marketplace

Deploy PMM Server with AWS Marketplace when you need quick setup with pre-configured settings, integrated billing, and enterprise-grade security for your existing AWS infrastructure.

## Prerequisites

Before you begin, ensure you have:

- an active AWS account with appropriate permissions
- IAM permissions to create and manage EC2 instances, storage, and 
- basic understanding of AWS VPC and security group concepts

## Plan your deployment

### Instance sizing recommendations

Choose your EC2 instance type based on your monitoring needs. Start with a smaller instance type and scale up as needed. You can resize EC2 instances with minimal downtime.

| Monitored hosts | Instance type | vCPUs | Memory | Storage |
|----------------|---------------|-------|--------|---------|
| 1-10 hosts     | t3.medium     | 2     | 4 GB   | 20 GB   |
| 10-50 hosts    | t3.large      | 2     | 8 GB   | 50 GB   |
| 50-200 hosts   | t3.xlarge     | 4     | 16 GB  | 100 GB  |
| 200+ hosts     | t3.2xlarge+   | 8+    | 32+ GB | 200+ GB |

### Storage planning

PMM Server stores all monitoring data in the `/srv` partition. Plan storage based on:

- number of monitored hosts
- retention period for collected data
- frequency of metric collection

As a reference, the [PMM Demo](https://pmmdemo.percona.com/) site consumes approximately 230 MB per host per day, which totals around 6.9 GB per host over a 30-day retention period.

For 50 hosts with 30-day retention: 50 × 6.9 GB = 345 GB minimum storage. 

For more information, see our blog post [How much disk space should I allocate for Percona Monitoring and Management](https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/).

### Storage recommendations

- include 20-30% buffer for unexpected spikes and growth
- use GP3 volumes for better price/performance than GP2
- consider higher IOPS for deployments with 100+ hosts

## Network and security planning

Plan your network configuration before deployment:

Required ports:
- port 22 (SSH) - for administrative access
- port 443 (HTTPS) - for PMM web interface
- port 3306 (MySQL) - if monitoring RDS instances

### Estimate costs

PMM Server software is free, but expect AWS infrastructure costs depending on your instance size and storage needs.

Use the [AWS Pricing Calculator](https://calculator.aws/#/) to estimate monthly costs based on your planned configuration.

## Deploy PMM Server

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
12. Assign a meaningful name to the instance to help distinguish it from others in your environment.

## Next steps


[Configure PMM Clients](../../../install-pmm-client/index.md) to start monitoring your infrastructure
[Register Client nodes](../../../register-client-node/index.md) with your PMM Server
[Configure SSL certificates](../../../../admin/security/ssl_encryption.md) for production use
[Set up monitoring alerts](../../../../alert/index.md)for proactive monitoring



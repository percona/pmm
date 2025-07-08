# Plan PMM Server deployment from AWS

Deploy PMM Server with AWS Marketplace when you need a quick setup with pre-configured settings, integrated billing, and enterprise-grade security for your existing AWS infrastructure.

## Prerequisites

Before you begin, ensure you have:

- an active AWS account with appropriate permissions
- IAM permissions to create and manage EC2 instances, storage, VPC and security groups
- understanding of AWS networking concepts (VPC, subnets, security group concepts)

## Choose the right instance size

Select an EC2 instance based on the number of hosts you plan to monitor. 

Start small and scale as your monitoring needs grow. EC2 instances can be resized with minimal downtime.

| Monitored hosts | Instance type | vCPUs | Memory | Storage |
|----------------|---------------|-------|--------|---------|
| 1-10 hosts     | t3.medium     | 2     | 4 GB   | 20 GB   |
| 10-50 hosts    | t3.large      | 2     | 8 GB   | 50 GB   |
| 50-200 hosts   | t3.xlarge     | 4     | 16 GB  | 100 GB  |
| 200+ hosts     | t3.2xlarge+   | 8+    | 32+ GB | 200+ GB |

### Plan storage 

PMM Server stores all monitoring data in the `/srv` partition. Plan storage based on the:

- number of monitored hosts
- retention period for collected data
- frequency of metric collection

As a reference, the [PMM Demo](https://pmmdemo.percona.com/) site consumes approximately 230 MB per host per day, which totals around 6.9 GB per host over a 30-day retention period.

For 50 hosts with 30-day retention: 50 × 6.9 GB = 345 GB minimum storage. 

### Storage recommendations

- include 20-30% buffer for unexpected spikes and growth
- use GP3 volumes for better price/performance than GP2
- consider higher IOPS for deployments with 100+ hosts

## Network and security planning

Plan your network configuration before deployment:

Required ports:

- port 22 (SSH) for administrative access
- port 443 (HTTPS) for PMM web interface
- port 3306 (MySQL) if monitoring RDS instances

## Estimate costs

PMM Server software is free, but plan for AWS infrastructure costs depending on your instance size and storage needs.

Use the [AWS pricing calculator](https://calculator.aws/#/) to estimate monthly costs based on your planned configuration.

## Plan backups

PMM Server uses a simple backup architecture - all monitoring data is stored in the `/srv` partition, which means you only need to back up one EBS volume to protect all your PMM data. This simplifies your backup strategy and reduces complexity.

When planning your deployment, consider that you'll need to create point-in-time snapshots of the EBS volume containing the `/srv` partition. Plan for snapshot storage costs and determine your backup frequency and retention requirements.

Follow the AWS documentation for [Create Amazon EBS snapshots](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-creating-snapshot.html) to understand the backup process you'll implement after deployment.

## Next steps

Once you've completed your planning:

- [Deploy PMM Server](../aws/deploy_aws.md) 
- [Configure security and access](../aws/configure_aws.md) 
- [Install PMM Client](../../../install-pmm-client/index.md)
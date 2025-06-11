# Deploy PMM Server on AWS

Deploy Percona Monitoring and Management (PMM) Server from AWS Marketplace to a running, accessible instance.

To install  PMM Server from AWS Marketplace:
{.power-number}

1. Go to [AWS Marketplace](https://aws.amazon.com/marketplace) and search for **Percona Monitoring and Management Server** or [access the PMM Server listing](https://aws.amazon.com/marketplace/pp/prodview-uww55ejutsnom) directly.

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

## Limit access to the instance

Configure security settings to restrict access to your PMM Server instance using AWS security groups and key pairs.
{.power-number}

1. In the **Security Group** section, which acts like a firewall, choose how you want to configure security group settings:

   === "Create new based on seller settings (recommended)"
       - Select the preselected option `Create new based on seller settings`
       - This creates a security group with recommended baseline settings
       - You can modify the rules after deployment as needed
       - Good starting point for most deployments

   === "Use existing security group"
       - Select an existing security group from your account
       - Ensure it includes the required ports listed below
       - Useful for integrating with existing security policies

2. In the **Key Pair** section, select an already set up EC2 key pair to enable secure SSH access to your instance. Without a valid key pair, you won't be able to SSH into your instance for administrative tasks. Ensure you have access to the private key file.

   ![AWS Key Pair](../../../../images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.3.png)


3.Ensure the security group allows communication via the following ports:

   ![Security Group Settings](../../../../images/aws-marketplace.pmm.launch-on-ec2.1-click-launch.2.png)

### Required ports
- Port 22 (SSH): Administrative access to the instance
- Port 80 (HTTP): Initial PMM web interface access
- Port 443 (HTTPS): Secure PMM web interface access
- Port 3306 (MySQL): If monitoring RDS instances directly

## Apply settings

Review your configuration, deploy the instance, and complete the initial setup in the EC2 Console.
{.power-number}

1. Scroll to the top of the page to review all your instance settings.

2. Click the **Launch with 1 click** button to deploy your PMM instance with the configured settings. Depending on your AWS Marketplace view, the launch button may be labeled as **Accept Software Terms & Launch with 1-Click**.

### Configure your instance in EC2 console

#### Access the EC2 console

After clicking **Launch with 1 click**, your PMM instance will begin deploying. To continue configuration:
{.power-number}

1. Click the **EC2 Console** link that appears at the top of the confirmation page.
2. Alternatively, navigate to the [EC2 Console](https://console.aws.amazon.com/ec2/) directly.

#### Locate and name your instance

Your new PMM instance will appear in the EC2 instances table with the following details:

- **Status**: Initially shows "Pending" while launching.
- **Name**: Empty by default (shows as "-").
- **Instance Type**: Matches your selected configuration
- **State**: "Running" once fully deployed.

![EC2 Console Instance List](../../../../images/aws-marketplace.ec2-console.pmm.1.png)

#### Monitor instance status

Monitor your instance deployment progress:

| Status | Description | Expected Duration |
|--------|-------------|-------------------|
| **Pending** | Instance is being created | 1-2 minutes |
| **Running** | Instance is active and accessible | Ready for use |
| **Status Checks** | System and instance checks | 2-5 minutes |

## Launch PMM Server

After deploying PMM Server from AWS Marketplace, access your PMM Server:
{.power-number}

1. Wait until the AWS console shows the instance is in "Running" state.

2. In the EC2 console, select your instance and copy its **IPv4 Public IP** in the instance details or the **Public IP** field from the **Properties** panel:

    ![Public IP Field](../../../../images/aws-marketplace.pmm.ec2.properties)

3. Open the IP address in a web browser and log into PMM using the default credentials `admin`/`your instance ID`.

    ![PMM Login](../../../../images/PMM_Login.png)

4. Change the default credentials then use the new ones on the PMM Server home page:

    ![PMM Home Dashboard](../../../../images/PMM_Home_Dashboard.png)

These credentials not only manage access to the PMM web interface but also facilitate authentication between the PMM Server and PMM Clients. You will need to reuse these credentials when configuring PMM Clients on other hosts.

### SSH access

For SSH access instructions, see [Connecting to Your Linux Instance Using SSH](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html). 

Replace the user name `ec2-user` with `admin`. You can also add SSH keys later through the **PMM Configuration > Settings > SSH Key** page.


!!! warning "Security critical"
    Always change the default password immediately. These credentials will be used for both web interface access and PMM Client authentication.


## Configure PMM server IP settings

### Configure PMM server to use a private IP only

By default, your EC2 instance will have a private IP for internal VPC network access. To use only the private IP:

=== "During EC2 instance creation"
    To use only the private IP for your EC2 instance during EC2 instance creation:
    {.power-number}

    1. In the **Network Settings** section, uncheck **Auto-assign public IP**.
    2. Do not assign an Elastic IP to the instance.

=== "For an existing instance"
    To use only the private IP for an existing instance:
    {.power-number}

    1. If a public IP is assigned, remove it by disassociating it in the EC2 console.
    2. If an Elastic IP is assigned, disassociate it from the instance.

### Access PMM server using only a private IP

To access your PMM Server using only a private IP:
{.power-number}

1. Ensure you're connected to your VPC.
2. Use the private IP address to access the PMM Server dashboard.

### Configure PMM server to use an Elastic IP (optional)

For a static, public-facing IP address:
{.power-number}

1. Allocate an Elastic IP address in the EC2 console.
2. Associate the Elastic IP address with your EC2 instance's network interface ID.

!!! note "Elastic IP considerations"
    Associating a new Elastic IP to an instance with an existing Elastic IP will disassociate the old one, but it will remain allocated to your account.

For detailed information on EC2 instance IP addressing, see the [AWS documentation on using instance addressing](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-instance-addressing.html).

## Resize the EBS volume

To increase available disk space:
{.power-number}

1. Your AWS instance comes with a predefined size which can become a limitation. To make more disk space available to your instance, increase the size of the EBS volume as needed. For instructions, see [Modifying the Size, IOPS, or Type of an EBS Volume on Linux](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-modify-volume.html).

2. After updating the EBS volume, PMM Server will auto-detect changes within approximately 5 minutes and reconfigure itself.

!!! tip "Auto-detection"
    PMM Server automatically detects EBS volume changes and reconfigures itself within 5 minutes. No manual intervention is required.

## Remove PMM server from AWS

To remove PMM Server:
{.power-number}

1. Find the instance in the EC2 Console:

    ![EC2 Console](../../../../images/aws-marketplace.pmm.ec2.remove1.png)

2. Select **Instance state** menu and **Terminate instance**:

    ![Terminate Instance](../../../../images/aws-marketplace.pmm.ec2.remove2.png)

3. Confirm termination operation:

    ![Confirm Terminate](../../../../images/aws-marketplace.pmm.ec2.remove3.png)

!!! warning "Data loss warning"
    Terminating an instance permanently deletes all data stored on the instance. Ensure you have created backups before termination.

Once your instance status shows "Running" and passes all status checks:
{.power-number}

1. Note the Public IP for PMM access.
2. Configure DNS (optional) to set up a custom domain name.
3. Access the PMM web interface at `http://your-instance-ip`.
4. Consider configuring SSL/TLS for production use.

!!! warning "Security reminder"
    Your PMM instance is now accessible via the internet. Ensure your security group settings restrict access to trusted IP addresses only.

## Next steps
- [Configure PMM server](../aws/configure_aws.md) for security and authentication
- [Configure PMM clients](../../../install-pmm-client/index.md) to start monitoring your infrastructure
- [Register client nodes](../../../register-client-node/index.md) with your PMM Server
- [Improve PMM EC2 instance resilience using CloudWatch Alarm actions](https://www.percona.com/blog/2021/04/29/improving-percona-monitoring-and-management-ec2-instance-resilience-using-cloudwatch-alarm-actions/)
- [Simplify use of ENV eariables in PMM AMI](https://www.percona.com/blog/simplify-use-of-env-variables-in-percona-monitoring-and-management-ami/)
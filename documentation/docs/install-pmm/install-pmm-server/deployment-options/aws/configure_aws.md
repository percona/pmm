# Configure PMM server on AWS

Complete the essential security configuration, user management, and ongoing maintenance for your PMM Server deployment on AWS.

## Prerequisites

Before configuring your PMM Server, ensure you have:
- Completed [planning your PMM Server deployment](../aws/plan_aws.md) including instance sizing, storage, and network requirements
- Successfully [deployed PMM Server from AWS Marketplace](../aws/deploy_aws.md) 
- Completed the [initial login and changed default credentials](../aws/deploy_aws.md#initial-pmm-server-access)
- Your PMM Server instance is running and accessible via HTTPS

## Configure SSL/TLS

Replace the self-signed certificate with a proper SSL certificate for production.

=== "Let's Encrypt certificate (free)"
    {.power-number}

    1. Make sure that the domain name pointing to your PMM Server IP.
    2. Check that port 80 temporarily open for certificate validation.
    3. Install and configure:
    ```bash
    # Install certbot
    sudo apt update
    sudo apt install certbot

    # Stop PMM temporarily
    sudo docker stop pmm-server

    # Obtain certificate (replace yourdomain.com)
    sudo certbot certonly --standalone -d pmm.yourdomain.com

    # Configure PMM to use the certificate
    sudo cp /etc/letsencrypt/live/pmm.yourdomain.com/fullchain.pem /srv/pmm-certs/certificate.crt
    sudo cp /etc/letsencrypt/live/pmm.yourdomain.com/privkey.pem /srv/pmm-certs/certificate.key
    sudo chown pmm:pmm /srv/pmm-certs/certificate.*
    sudo chmod 600 /srv/pmm-certs/certificate.*

    # Restart PMM Server
    sudo docker start pmm-server
    ```

=== "Commercial certificate"
    If you have a commercial SSL certificate:
    {.power-number}

    1. Upload certificate files:
       ```bash
       scp -i /path/to/your-key.pem certificate.crt admin@<instance-ip>:/tmp/
       scp -i /path/to/your-key.pem private.key admin@<instance-ip>:/tmp/
       ```

    2. Install certificates:
       ```bash
       sudo mv /tmp/certificate.crt /srv/pmm-certs/
       sudo mv /tmp/private.key /srv/pmm-certs/certificate.key
       sudo chown pmm:pmm /srv/pmm-certs/certificate.*
       sudo chmod 600 /srv/pmm-certs/certificate.*
       sudo docker restart pmm-server
       ```

### Create additional users
{.power-number}

1. Access **PMM > Configuration > User Management**.
2. Click **Add User** and configure:
   - **Admin**: Full system access
   - **Editor**: Dashboard editing, no system config
   - **Viewer**: Read-only access

3. Limit access based on job responsibilities and use viewer accounts for stakeholders who only need to see metrics.

Use the principle of least privilege when assigning user roles. Most users only need Viewer access to see dashboards and metrics.

## Firewall configuration

Configure the OS-level firewall:

```sh
# SSH to PMM Server
ssh -i /path/to/your-key.pem admin@<your-instance-ip>

# Configure firewall rules
sudo ufw allow 22/tcp    # SSH access
sudo ufw allow 443/tcp   # HTTPS PMM interface
sudo ufw --force enable
```

## Generate API keys for automation
{.power-number}

1. Navigate to **PMM Configuration > API Keys**.
2. Create API keys with descriptive names and minimum required permissions.
3. Set appropriate expiration dates.
4. Store API keys securely in a password manager or secrets vault and rotate them regularly. Never commit them to version control or share them in plain text.

## PMM Client integration

### Server URL configuration

Configure the PMM Server URL for client connections:

=== "Public deployment"
    ```bash
    PMM_SERVER_URL="https://<elastic-ip-or-domain>:443"
    ```

=== "Private deployment"
    ```bash
    PMM_SERVER_URL="https://<private-ip>:443"
    ```

### Client authentication setup

PMM Client authentication uses the same credentials you set for the web interface:

```bash
# Example PMM Client configuration command
pmm-admin config --server-insecure-tls --server-url=https://admin:your-password@<pmm-server-ip>:443
```

#### Security considerations

- Use the web interface credentials you created
- Consider creating dedicated API users for client authentication
- Avoid putting passwords in command history (use environment variables)

### Test connection

Test PMM Client connectivity:

```bash
# Test PMM Server connectivity
curl -k https://<pmm-server-ip>:443/ping
# Expected response: "OK"

# Test API authentication
curl -k -u admin:your-password https://<pmm-server-ip>:443/v1/readyz
# Expected response: {"status":"ok"}
```

## AWS service integration

### RDS monitoring setup

To configure security groups for RDS access:
{.power-number}

1. Modify your RDS security group to add inbound rule: MySQL/Aurora (3306) from PMM security group.
2. Test connectivity:
   ```bash
   # From PMM Server
   nc -zv your-rds-endpoint.amazonaws.com 3306
   ```
3. Add RDS instance in PMM using the RDS endpoint hostname. 

### CloudWatch integration

To configure CloudWatch metrics export:
{.power-number}

1. Go to **PMM Configuration > Settings > Advanced Settings**.
2. Enable CloudWatch Integration with your AWS region.
3. Configure IAM role with CloudWatch permissions.

!!! note "IAM permissions"
    Ensure your PMM instance has an IAM role with CloudWatch permissions for metrics export and integration.

## Optimize memory allocation

To optimize memory allocation based on instance size:

```bash
# Check current memory usage
free -h
docker stats pmm-server

# For t3.medium (4GB RAM), adjust memory limits:
# Prometheus: 1GB, ClickHouse: 1GB, Grafana: 512MB
```
Scale memory allocations proportionally for larger instances.

## Enable PMM Server self-monitoring
{.power-number}

1. Go to **PMM Configuration > Settings > Advanced Settings**.
2. Enable Internal Monitoring with 30-day retention.
3. Monitor CPU, memory, disk I/O, and PMM service health.

## Set up CloudWatch alarms

```bash
# High CPU usage alarm
aws cloudwatch put-metric-alarm \
    --alarm-name "PMM-Server-High-CPU" \
    --alarm-description "PMM Server CPU over 80%" \
    --metric-name CPUUtilization \
    --namespace AWS/EC2 \
    --threshold 80 \
    --comparison-operator GreaterThanThreshold
```

## Update PMM Server

Always create a backup before updating PMM Server. Test updates in a development environment first.

```bash
# Create backup before updating
sudo docker exec pmm-server pmm-admin summary > /tmp/pmm-config-backup.txt

# Update PMM Server
sudo docker stop pmm-server
sudo docker rm pmm-server
sudo docker pull percona/pmm-server:latest
sudo docker run -d -p 80:80 -p 443:443 --volumes-from pmm-data --name pmm-server --restart always percona/pmm-server:latest

# Verify update
curl -k https://localhost:443/v1/version
```

## Create manual backup

```bash
# Get volume ID and create snapshot
INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
VOLUME_ID=$(aws ec2 describe-instances --instance-ids $INSTANCE_ID --query 'Reservations[0].Instances[0].BlockDeviceMappings[?DeviceName==`/dev/sdf`].Ebs.VolumeId' --output text)

aws ec2 create-snapshot \
    --volume-id $VOLUME_ID \
    --description "PMM Server manual backup $(date +%Y-%m-%d_%H:%M:%S)"
```
![AWS Marketplace](../../../../images/aws-marketplace.png)

## Restore from backup

To restore PMM Server from a backup:
{.power-number}

1. Create a new volume using the latest snapshot of the PMM data volume:

![Create Volume](../../../../images/aws-marketplace.pmm.ec2.backup2.png)

2. Stop the PMM Server instance.

3. Detach the current PMM data volume:

![Detach Volume](../../../../images/aws-marketplace.pmm.ec2.backup3.png)

4. Attach the new volume:

![Attach Volume](../../../../images/aws-marketplace.pmm.ec2.backup4.png)

5. Start the PMM Server instance.

!!! note "Recovery time"
    The restore process typically takes 5-15 minutes depending on volume size and AWS region performance.

## Terminate instance

{.power-number}

1. Create final backup:
   ```bash
   aws ec2 create-snapshot --volume-id $DATA_VOLUME_ID --description "Final backup before termination"
    ```
2. Disconnect all PMM clients:
   ```bash
    # On each monitored server
    pmm-admin remove --all
    ```
3. Export configuration:
   ```bash
    sudo docker exec pmm-server pmm-admin summary > pmm-final-config.txt
    ```
4. Stop PMM services:
   ```bash
   sudo docker stop pmm-server
   ```
5. Terminate the instance:
   ```bash
   aws ec2 terminate-instances --instance-ids i-1234567890abcdef0
   ```
6. Clean up resources:

   ```bash
   # Release Elastic IP if using one
   aws ec2 release-address --allocation-id eipalloc-12345678\
   ```
!!! danger alert alert-danger "Data loss warning"
    Instance termination permanently deletes all data. Ensure you have completed all backup procedures before proceeding.

## Next steps

With your PMM Server fully configured and secured:
{.power-number}

- [Configure PMM clients](../../../install-pmm-client/index.md) to start monitoring your infrastructure
- [Register client nodes](../../../register-client-node/index.md) with your PMM Server
- [Configure SSL certificates](../../../../admin/security/ssl_encryption.md) for production use
- [Set up monitoring alerts](../../../../alert/index.md) for proactive monitoring



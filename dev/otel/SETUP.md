# OpenTelemetry Logging Setup Instructions (PoC)

## Project Structure
This project has the following directory structure:

```
├── clickhouse
│   ├── config.d
│   │   └── config-override.xml
│   ├── client-config.xml
│   ├── config-override.xml
│   ├── generate-certs.sh
│   └── test.sql
├── doc
│   ├── otel-collector.png
│   ├── password-change-failure-alert.png
│   └── password-change-success-alert.png
├── grafana
│   ├── alert-rules.yml
│   ├── change-admin-password
│   ├── clickhouse-datasource.yml
│   ├── contact-points.yml
│   ├── datasources.yml
│   └── notification-policies.yml
├── nginx
│   └── nginx.conf
├── test
│   ├── clickhouse-test.sh
│   └── setup-test.sh
├── .env.example
├── config.yml
├── docker-compose.yml
├── README.md
└── SETUP.md
```

## Setup Steps

### 1. Clone the Project
```bash
git clone https://github.com/percona/pmm.git
cd pmm/dev/otel
```

### 2. Configure Environment
```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file with your settings
vim .env  # or use your preferred editor
```

**Required Environment Variables:**
- `GF_SMTP_FROM_ADDRESS`: Email address for sending alert notifications
- `GF_SECURITY_ADMIN_EMAIL`: Admin email address for Grafana (for sending user invites, etc.)

**Example .env configuration:**
```bash
# Email configuration for Grafana SMTP notifications (required)
GF_SMTP_FROM_ADDRESS=admin@yourcompany.com
GF_SECURITY_ADMIN_EMAIL=security@yourcompany.com
```

### 3. Update Email Addresses for Alerts
Edit the contact points configuration to use your email addresses:
```bash
# Edit the contact points file
vim grafana/contact-points.yml

# Update the addresses with your emails:
# addresses: "admin@yourcompany.com;security@yourcompany.com"
```

### 4. Start the Environment
```bash
# Start all services
docker compose up -d

# Check service status
docker compose ps

# View logs
docker compose logs -f cert-generator
docker compose logs -f otel-collector
docker compose logs -f pmm-server
```

### 5. Generate Logs
PMM generates quite some logs on during user interaction, so after a few moments of interaction, you can start exploring the logs. However, you may choose to generate a few log lines manually, for example Nginx logs, for testing purposes:

```bash
# Generate various HTTP responses
curl -k -u admin:admin https://localhost/                    # 200 OK
curl -k -u admin:admin https://localhost/graph/api/users     # 200 OK
curl -k                https://localhost/graph/api/users/1   # 401 Unauthorized
curl -k -u admin:admin https://localhost/graph/nonexistent   # 404 Not Found
```

### 6. Access ClickHouse
```bash
# Connect to ClickHouse CLI
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel
```

### 7. Run Test Queries
Execute the test queries from the `clickhouse/test.sql` file in the ClickHouse client.

### 8. Test Security Alerts
To test the admin password change alert, you can change the admin password in Grafana. This will trigger an alert if configured correctly:

```bash
# Use the command line tool:
docker exec -it pmm-server change-admin-password "<new-password>"
```

**Expected behavior:**
- The alert should trigger within 1 minute of password change
- You should receive an email notification delivered to the configured addresses
- Check MailHog UI at http://localhost:8025 to see emails sent by triggered alerts

Likewise, to test the admin password change failure alert, you can pass an empty password or one that does not meet the password requirements:

```bash
# Use the command line tool with an empty password
docker exec -it pmm-server change-admin-password ""
```

### 9. Monitor Alert System
```bash
# Check Grafana alerting logs
docker exec -it pmm-server bash
grep "ngalert" /srv/logs/grafana.log

# View alert rules in Grafana UI
# Go to https://localhost:443
# Navigate to Alerting > Alert Rules

# Check contact points and notification policies
# Navigate to Alerting > Contact Points
# Navigate to Alerting > Notification Policies
```

### 10. Adding more Logs
You can add more log sources to PMM server by modifying the `config.yml` file. If you want to add an external log source, you can configure the OpenTelemetry Collector to scrape logs from that source. To read more, refer to the [OpenTelemetry Collector documentation](https://opentelemetry.io/docs/collector/configuration).

### 11. Changing Log Retention
To change the log retention period, modify the ClickHouse table TTL settings in the `clickhouse/config.d/config-override.xml` file:
```xml
<clickhouse>
    <profiles>
        <default>
            <ttl_only_drop_parts>1</ttl_only_drop_parts>
        </default>
    </profiles>
    <tables>
        <otel.logs>
            <ttl>
                <column name="TimestampTime" unit="day" value="7"/>
            </ttl>
        </otel.logs>
    </tables>
</clickhouse>
```

Alternatively, you can also change the TTL by running the following query using the ClickHouse client:
```sql
ALTER TABLE otel.logs MODIFY TTL TimestampTime + INTERVAL 7 DAY;
```

### 12. Creating Dashboards
You can create custom dashboards in PMM to visualize the logs. Use the `ClickHouse-Logs` data source to query the `otel.logs` table and create panels for different log types, such as Nginx access logs, Grafana logs, pmm-managed logs, pmm-agent logs, and more.

#### Example Query for Log Linecount by Service
Panel Description: Log Linecount by Service
```sql
SELECT ServiceName AS service, COUNT(*) as count FROM "otel"."logs" WHERE ( timestamp >= $__fromTime AND timestamp <= $__toTime ) GROUP BY service ORDER BY count DESC;
```

#### Example Query for Nginx Status by Severity
Panel Description: Nginx Status by Severity
```sql
SELECT
CASE WHEN LogAttributes['status'] = '' THEN 'N/A' ELSE LogAttributes['status'] END AS mapping, COUNT(*) as count
FROM otel.logs
WHERE ( Timestamp >= $__fromTime AND Timestamp <= $__toTime ) AND ServiceName = 'nginx'
GROUP BY LogAttributes['status'], SeverityNumber
ORDER BY LogAttributes['status']
```

#### Example Query for General Logs
Panel Description: General Logs
```sql
SELECT Timestamp as "timestamp", Body as "body", SeverityText as "level", LogAttributes as "labels", TraceId as "traceID" FROM "otel"."logs" WHERE ( timestamp >= $__fromTime AND timestamp <= $__toTime ) ORDER BY timestamp DESC LIMIT 1000
```

## Troubleshooting

### Check Project Setup
```bash
cd test
bash setup-test.sh
```

### ClickHouse Data Verification
```bash
# Check table exists and has data
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel -q "SELECT count() FROM otel.logs"

# View most recent logs
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel -q "SELECT * FROM otel.logs ORDER BY Timestamp DESC LIMIT 10"
```

## Services and Ports

- **PMM**: https://localhost:443
- **ClickHouse Native**: localhost:9000
- **OpenTelemetry OTLP gRPC**: localhost:4317
- **OpenTelemetry OTLP HTTP**: localhost:4318
- **MailHog Web UI**: http://localhost:8025 (for testing email notifications)

## Alert System Configuration

This PoC includes a complete security alerting system that monitors:

### Security Alerts:
- **Admin Password Changes**: Detects when admin password is successfully reset
- **Failed Password Attempts**: Detects failed admin password change attempts

### Alert Configuration Files:
- `grafana/alert-rules.yml`: Defines the alert rules and queries
- `grafana/contact-points.yml`: Email notification configuration
- `grafana/notification-policies.yml`: Alert routing and grouping policies
- `grafana/datasources.yml`: ClickHouse data source for log queries

### Notification Flow:
1. OpenTelemetry Collector ingests Grafana logs
2. Logs are stored in ClickHouse `otel.logs` table
3. Grafana alert rules query ClickHouse for security events
4. Alerts are routed via notification policies
5. Email notifications are sent via configured SMTP (MailHog for testing)

## Cleanup
```bash
# Stop and remove all containers (keep the data)
docker compose down

# Stop and remove all container and volumes (this will delete all data)
docker compose down -v

# Remove images
docker compose down --rmi all
```
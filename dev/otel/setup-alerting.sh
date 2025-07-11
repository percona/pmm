#!/bin/bash

# Setup script for Grafana alerting with OpenTelemetry logs
# This script helps configure email notifications and starts the services

set -e

echo "🚀 Setting up Grafana Alerting with OpenTelemetry Logs..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found. Please create it with the required variables."
    exit 1
fi

# Source the .env file
source .env

# Check required environment variables
if [ -z "$GF_SMTP_FROM_ADDRESS" ]; then
    echo "❌ GF_SMTP_FROM_ADDRESS is not set in .env file"
    exit 1
fi

if [ -z "$GF_SECURITY_ADMIN_EMAIL" ]; then
    echo "❌ GF_SECURITY_ADMIN_EMAIL is not set in .env file"
    exit 1
fi

echo "✅ Environment variables validated"

# Update notification channels with real email addresses
echo "📧 Updating notification channels configuration..."
sed -i.bak "s/admin@yourcompany.com;security@yourcompany.com/$GF_SECURITY_ADMIN_EMAIL/g" grafana-notification-channels.yml

# Check if docker-compose.yml exists
if [ ! -f docker-compose.yml ]; then
    echo "❌ docker-compose.yml file not found"
    exit 1
fi

# Check if config.yml exists
if [ ! -f config.yml ]; then
    echo "❌ OpenTelemetry config.yml file not found"
    exit 1
fi

echo "✅ Configuration files validated"

# Create necessary directories for logs
echo "📁 Creating log directories..."
sudo mkdir -p /srv/logs
sudo chmod 755 /srv/logs

# Create sample log files if they don't exist
if [ ! -f /srv/logs/nginx.log ]; then
    echo "📝 Creating sample nginx access log..."
    sudo tee /srv/logs/nginx.log > /dev/null <<EOF
{"timestamp":"$(date -u +%Y-%m-%dT%H:%M:%S%z)","status":200,"method":"GET","uri":"/","remote_addr":"127.0.0.1","user_agent":"Mozilla/5.0"}
{"timestamp":"$(date -u +%Y-%m-%dT%H:%M:%S%z)","status":401,"method":"POST","uri":"/admin/login","remote_addr":"192.168.1.100","user_agent":"curl/7.68.0"}
EOF
fi

if [ ! -f /srv/logs/nginx-error.log ]; then
    echo "📝 Creating sample nginx error log..."
    sudo tee /srv/logs/nginx-error.log > /dev/null <<EOF
$(date '+%Y/%m/%d %H:%M:%S') [error] 1234#0: *1 access forbidden by rule, client: 192.168.1.100, server: localhost, request: "GET /admin HTTP/1.1", host: "localhost"
$(date '+%Y/%m/%d %H:%M:%S') [warn] 1234#0: *2 upstream server temporarily disabled, client: 192.168.1.50, server: localhost, request: "POST /api/login HTTP/1.1", host: "localhost"
EOF
fi

if [ ! -f /srv/logs/grafana.log ]; then
    echo "📝 Creating sample grafana log..."
    sudo tee /srv/logs/grafana.log > /dev/null <<EOF
t=$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ) level=info msg="User login successful" user=admin
t=$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ) level=warn msg="Failed login attempt" user=admin client_ip=192.168.1.100
t=$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ) level=error msg="Admin password change requested" user=admin action=password_change
EOF
fi

echo "✅ Sample log files created"

# Build and start the services
echo "🐳 Starting services with Docker Compose..."
docker-compose down --remove-orphans 2>/dev/null || true
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are running
echo "🔍 Checking service health..."
if ! docker-compose ps | grep -q "pmm-server.*Up"; then
    echo "❌ PMM Server is not running"
    docker-compose logs pmm-server
    exit 1
fi

if ! docker-compose ps | grep -q "otel-collector.*Up"; then
    echo "❌ OpenTelemetry Collector is not running"
    docker-compose logs otel-collector
    exit 1
fi

echo "✅ Services are running successfully"

# Display access information
echo ""
echo "🎉 Setup completed successfully!"
echo ""
echo "📊 Access Information:"
echo "   - PMM Server (Grafana): https://localhost:443"
echo "   - ClickHouse: localhost:9000"
echo "   - MailHog (Email Testing): http://localhost:8025"
echo "   - OpenTelemetry Collector Metrics: http://localhost:8888/metrics"
echo ""
echo "🔐 Default Credentials:"
echo "   - Username: admin"
echo "   - Password: admin"
echo ""
echo "📧 Email Configuration:"
echo "   - SMTP is configured to use MailHog for testing"
echo "   - Check http://localhost:8025 for sent emails"
echo "   - Update .env file with real SMTP settings for production"
echo ""
echo "🚨 Alert Rules:"
echo "   - Admin password change detection"
echo "   - Multiple failed login attempts"
echo "   - Security event tagging"
echo ""
echo "📝 Next Steps:"
echo "   1. Login to Grafana at https://localhost:443"
echo "   2. Check the 'Security' folder for alert rules"
echo "   3. Test alerts by triggering security events in logs"
echo "   4. Configure real email/Slack notifications in production"
echo ""
echo "🔧 Troubleshooting:"
echo "   - View logs: docker-compose logs [service-name]"
echo "   - Check collector config: docker-compose exec otel-collector cat /etc/otel/config.yml"
echo "   - Test ClickHouse: docker-compose exec pmm-server clickhouse-client"
echo ""

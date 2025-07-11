#!/bin/bash

# This script helps configure email notifications and starts the services

set -e

# Check if services are running
echo "🔍 Checking service health..."
if ! docker compose ps | grep -q "pmm-server.*Up"; then
    echo "❌ PMM Server is not running"
    docker compose logs pmm-server
    exit 1
fi

if ! docker compose ps | grep -q "otel-collector.*Up"; then
    echo "❌ OpenTelemetry Collector is not running"
    docker compose logs otel-collector
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
echo "   - Admin password change"
echo "   - Admin password change failure"
echo ""
echo "📝 Next Steps:"
echo "   1. Login to Grafana at https://localhost with the default credentials"
echo "   2. Check the 'Security' folder for alert rules"
echo "   3. Test alerts by changing the admin password or failing to do so (use change-admin-password script)"
echo "   4. Configure real email/Slack/etc notifications in production"
echo ""
echo "🔧 Troubleshooting:"
echo "   - View logs: docker compose logs [service-name]"
echo "   - Check collector config: config.yml"
echo "   - Test ClickHouse: docker compose exec pmm-server clickhouse-client --password=clickhouse --query 'SELECT 1'"
echo ""

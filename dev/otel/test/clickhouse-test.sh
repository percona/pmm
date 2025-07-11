#!/bin/bash

echo "Testing HTTP status code to OpenTelemetry severity mapping..."

# Wait for logs to be processed
echo "Waiting for logs to be processed..."
sleep 5

# Check the mapping results
echo "Checking severity mapping results from ClickHouse..."
docker exec otel-clickhouse clickhouse-client --user=default --password=clickhouse --query "
SELECT 
    'Status Code Mapping Summary:' AS summary
UNION ALL
SELECT 
    CONCAT(
        'HTTP ', CASE WHEN LogAttributes['status'] = '' THEN 'N/A' ELSE LogAttributes['status'] END, 
        ' -> ', SeverityText, 
        ' (', toString(SeverityNumber), ')'
    ) AS mapping
FROM otel.logs
WHERE Timestamp > now() - INTERVAL 5 MINUTE
GROUP BY LogAttributes['status'], SeverityText, SeverityNumber
ORDER BY LogAttributes['status']
"

echo ""
echo "Expected mappings:"
echo "- HTTP 2xx -> INFO (9)"
echo "- HTTP 4xx -> WARN (13)"
echo "- HTTP 5xx -> ERROR (17)"

# Check the log count within the last 5 minutes
echo ""
echo "Checking log count in ClickHouse for the last 5 minutes..."
docker exec otel-clickhouse clickhouse-client --user=default --password=clickhouse --query "
SELECT count() FROM otel.logs WHERE Timestamp >= now() - INTERVAL 5 MINUTE
"

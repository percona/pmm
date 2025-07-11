#!/bin/bash

echo "Testing HTTP status code to OpenTelemetry severity mapping..."

# Generate requests that will produce different status codes
echo "Generating test requests..."
curl -s http://localhost:8080/ > /dev/null  # 200 OK
curl -s http://localhost:8080/missing > /dev/null  # 404 Not Found
echo "Requests sent"

# Wait for logs to be processed
echo "Waiting for logs to be processed..."
sleep 5

# Check the mapping results
echo "Checking severity mapping results from ClickHouse..."
docker exec otel-clickhouse clickhouse-client --user=default --password=clickhouse --query "
SELECT 
    'Status Code Mapping Summary:' as summary
UNION ALL
SELECT 
    CONCAT(
        'HTTP ', LogAttributes['status'], 
        ' -> ', SeverityText, 
        ' (', toString(SeverityNumber), ')'
    ) as mapping
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

# Check the logs within the last 5 minutes
echo ""
echo "Checking logs in ClickHouse for the last 5 minutes..."
docker exec otel-clickhouse clickhouse-client --user=default --password=clickhouse --query "
SELECT count() FROM logs WHERE timestamp >= now() - INTERVAL 5 MINUTE
"

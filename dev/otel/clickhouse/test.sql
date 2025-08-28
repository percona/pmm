-- Test queries for the OpenTelemetry ClickHouse exporter auto-generated schema

-- Sample queries for the OpenTelemetry ClickHouse exporter auto-generated schema

-- 1. Basic log query with service filtering
SELECT 
    Timestamp,
    ServiceName,
    SeverityText,
    SeverityNumber,
    LogAttributes['status'] as status,
    LogAttributes['request_method'] as method,
    LogAttributes['request'] as uri,
    Body
FROM otel.logs 
WHERE ServiceName = 'nginx'
ORDER BY Timestamp DESC 
LIMIT 10;

-- 2. Error analysis by severity level
SELECT 
    Timestamp,
    ServiceName,
    SeverityText,
    SeverityNumber,
    LogAttributes['status'] as status,
    LogAttributes['remote_addr'] as client_ip,
    LogAttributes['request'] as request_uri
FROM otel.logs 
WHERE ServiceName = 'nginx' AND SeverityText IN ('error', 'fatal', 'warn')
ORDER BY Timestamp DESC 
LIMIT 100;

-- 3. Service overview by severity
SELECT 
    ServiceName,
    ResourceAttributes['service.version'] as service_version,
    ResourceAttributes['environment'] as environment,
    SeverityText,
    count() as log_count,
    min(Timestamp) as first_log,
    max(Timestamp) as last_log
FROM otel.logs 
GROUP BY ServiceName, service_version, environment, SeverityText
ORDER BY log_count DESC;

-- 4. Log volume over time (last 24 hours, grouped by hour)
SELECT 
    toStartOfHour(Timestamp) as hour,
    SeverityText,
    count() as log_count
FROM otel.logs 
WHERE Timestamp >= now() - INTERVAL 24 HOUR
GROUP BY hour, SeverityText
ORDER BY hour DESC, SeverityText;

-- 5. Real-time monitoring query using severity
SELECT 
    toStartOfHour(Timestamp) as hour,
    ServiceName,
    countIf(SeverityText = 'info') as info_logs,
    countIf(SeverityText = 'warn') as warn_logs,
    countIf(SeverityText = 'error') as error_logs,
    countIf(SeverityText = 'fatal') as fatal_logs,
    countIf(toUInt16OrZero(LogAttributes['status']) >= 400) as http_errors,
    avg(toFloat64OrZero(LogAttributes['request_time'])) as avg_response_time
FROM otel.logs 
WHERE Timestamp >= now() - INTERVAL 1 HOUR
GROUP BY hour, ServiceName
ORDER BY hour DESC;

-- 6. Error rates (4xx and 5xx responses)
SELECT 
    toStartOfHour(Timestamp) as hour,
    CASE 
        WHEN toUInt16OrZero(LogAttributes['status']) BETWEEN 400 AND 499 THEN '4xx_errors'
        WHEN toUInt16OrZero(LogAttributes['status']) BETWEEN 500 AND 599 THEN '5xx_errors'
        ELSE 'other'
    END as error_category,
    count() as error_count,
    round((count() * 100.0) / (
        SELECT count() 
        FROM otel.logs 
        WHERE LogAttributes['status'] != ''
        AND toUInt16OrZero(LogAttributes['status']) > 0 
        AND Timestamp >= now() - INTERVAL 24 HOUR
    ), 2) as error_percentage
FROM otel.logs 
WHERE LogAttributes['status'] != ''
    AND toUInt16OrZero(LogAttributes['status']) >= 400 
    AND Timestamp >= now() - INTERVAL 24 HOUR
    AND ServiceName = 'nginx'
GROUP BY hour, error_category
ORDER BY hour DESC, error_category;

-- 7. Successful request rates (2xx and 3xx responses)
SELECT 
    toStartOfHour(Timestamp) as hour,
    CASE 
        WHEN toUInt16OrZero(LogAttributes['status']) BETWEEN 200 AND 299 THEN '2xx_success'
        WHEN toUInt16OrZero(LogAttributes['status']) BETWEEN 300 AND 399 THEN '3xx_redirect'
    END as success_category,
    count() as success_count,
    round((count() * 100.0) / (
        SELECT count() 
        FROM otel.logs 
        WHERE LogAttributes['status'] != ''
          AND toUInt16OrZero(LogAttributes['status']) > 0 
          AND Timestamp >= now() - INTERVAL 24 HOUR
          AND ServiceName = 'nginx'
    ), 2) as success_percentage
FROM otel.logs 
WHERE LogAttributes['status'] != ''
    AND toUInt16OrZero(LogAttributes['status']) BETWEEN 200 AND 399
    AND Timestamp >= now() - INTERVAL 24 HOUR
    AND ServiceName = 'nginx'
GROUP BY hour, success_category
ORDER BY hour DESC, success_category;

-- 11. Overall request statistics
SELECT 
    count() as total_requests,
    countIf(toUInt16OrZero(LogAttributes['status']) BETWEEN 200 AND 299) as success_2xx,
    countIf(toUInt16OrZero(LogAttributes['status']) BETWEEN 300 AND 399) as redirect_3xx,
    countIf(toUInt16OrZero(LogAttributes['status']) BETWEEN 400 AND 499) as client_error_4xx,
    countIf(toUInt16OrZero(LogAttributes['status']) BETWEEN 500 AND 599) as server_error_5xx,
    round(avg(toFloat64OrZero(LogAttributes['request_time'])), 3) as avg_response_time,
    round(sum(toUInt64OrZero(LogAttributes['body_bytes_sent'])) / 1024 / 1024, 2) as total_mb_sent
FROM otel.logs 
WHERE LogAttributes['status'] != ''
    AND toUInt16OrZero(LogAttributes['status']) > 0 
    AND ServiceName = 'nginx'
    AND Timestamp >= now() - INTERVAL 24 HOUR;

-- 12. Top requested URIs
SELECT 
    LogAttributes['request'] as request_uri,
    count() as request_count,
    countIf(toUInt16OrZero(LogAttributes['status']) BETWEEN 200 AND 299) as success_count,
    countIf(toUInt16OrZero(LogAttributes['status']) >= 400) as error_count,
    round(avg(toFloat64OrZero(LogAttributes['request_time'])), 3) as avg_response_time
FROM otel.logs 
WHERE LogAttributes['request'] != ''
    AND Timestamp >= now() - INTERVAL 24 HOUR
    AND ServiceName = 'nginx'
GROUP BY request_uri
ORDER BY request_count DESC
LIMIT 10;

-- 14. Error log analysis
SELECT 
    toStartOfHour(Timestamp) as hour,
    SeverityText,
    LogAttributes['log_level'] as log_level,
    count() as error_count,
    groupArray(LogAttributes['message']) as sample_messages
FROM otel.logs 
WHERE SeverityText IN ('ERROR', 'FATAL', 'WARN') 
    AND Timestamp >= now() - INTERVAL 24 HOUR
    AND ServiceName = 'nginx'
GROUP BY hour, SeverityText, log_level
ORDER BY hour DESC, SeverityText;

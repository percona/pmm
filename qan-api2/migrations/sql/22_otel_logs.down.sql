-- Rollback OTEL logs tables
DROP VIEW IF EXISTS logs_by_service_hourly_mv;
DROP TABLE IF EXISTS logs_by_service_hourly;
DROP TABLE IF EXISTS otel_logs;


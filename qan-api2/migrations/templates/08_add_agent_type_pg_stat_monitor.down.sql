ALTER TABLE `metrics` MODIFY COLUMN `agent_type` Enum8(
    'qan-agent-type-invalid'=0,
    'qan-mysql-perfschema-agent'=1,
    'qan-mysql-slowlog-agent'=2,
    'qan-mongodb-profiler-agent'=3,
    'qan-postgresql-pgstatements-agent'=4
    ) COMMENT 'Agent Type that collect of metrics: slowlog, perf schema, etc.';

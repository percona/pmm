ALTER TABLE metrics MODIFY COLUMN `agent_type` Enum8(
  'agent_type_invalid' = 0,
  'mysql-perfschema' = 1,
  'mysql-slowlog' = 2,
  'mongodb-profiler' = 3,
  'postgresql-pgstatstatements' = 4
  ) COMMENT 'Agent Type that collect of metrics: slowlog, perf schema, etc.';
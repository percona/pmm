ALTER TABLE metrics
  ADD COLUMN `m_wal_bytes_cnt` Float32,
  ADD COLUMN `m_wal_bytes_sum` Float32 COMMENT 'Total bytes of WAL (Write-ahead logging) records',
  ADD COLUMN `m_plan_time_cnt` Float32 COMMENT 'Count of plan time.',
  ADD COLUMN `m_plan_time_sum` Float32 COMMENT 'Sum of plan time.',
  ADD COLUMN `m_plan_time_min` Float32 COMMENT 'Min of plan time.',
  ADD COLUMN `m_plan_time_max` Float32 COMMENT 'Max of plan time.';

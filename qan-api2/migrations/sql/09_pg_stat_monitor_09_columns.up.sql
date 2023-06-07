ALTER TABLE metrics
  ADD COLUMN `m_plans_calls_cnt` Float32,
  ADD COLUMN `m_plans_calls_sum` Float32 COMMENT 'Total number of planned calls',
  ADD COLUMN `m_wal_records_cnt` Float32,
  ADD COLUMN `m_wal_records_sum` Float32 COMMENT 'Total number of WAL (Write-ahead logging) records',
  ADD COLUMN `m_wal_fpi_cnt` Float32,
  ADD COLUMN `m_wal_fpi_sum` Float32 COMMENT 'Total number of FPI (full page images) in WAL (Write-ahead logging) records';

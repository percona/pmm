ALTER TABLE metrics
  DROP COLUMN `m_plans_calls_cnt`,
  DROP COLUMN `m_plans_calls_sum`,
  DROP COLUMN `m_wal_records_cnt`,
  DROP COLUMN `m_wal_records_sum`,
  DROP COLUMN `m_wal_fpi_cnt`,
  DROP COLUMN `m_wal_fpi_sum`;

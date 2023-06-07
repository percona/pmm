ALTER TABLE metrics
  DROP COLUMN `m_wal_bytes_cnt`,
  DROP COLUMN `m_wal_bytes_sum`,
  DROP COLUMN `m_plan_time_cnt`,
  DROP COLUMN `m_plan_time_sum`,
  DROP COLUMN `m_plan_time_min`,
  DROP COLUMN `m_plan_time_max`;

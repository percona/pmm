ALTER TABLE metrics
  DROP COLUMN m_wal_buffers_full_cnt,
  DROP COLUMN m_wal_buffers_full_sum,
  DROP COLUMN m_parallel_workers_to_launch_cnt,
  DROP COLUMN m_parallel_workers_to_launch_sum,
  DROP COLUMN m_parallel_workers_launched_cnt,
  DROP COLUMN m_parallel_workers_launched_sum;
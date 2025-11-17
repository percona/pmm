ALTER TABLE metrics
  ADD COLUMN m_wal_buffers_full_cnt Float32,
  ADD COLUMN m_wal_buffers_full_sum Float32 COMMENT 'Total number of times WAL buffers become full',
  ADD COLUMN m_parallel_workers_to_launch_cnt Float32,
  ADD COLUMN m_parallel_workers_to_launch_sum Float32 COMMENT 'Total number of parallel workers to launch',
  ADD COLUMN m_parallel_workers_launched_cnt Float32,
  ADD COLUMN m_parallel_workers_launched_sum Float32 COMMENT 'Total number of parallel workers launched';

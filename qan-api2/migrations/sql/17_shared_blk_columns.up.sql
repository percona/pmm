ALTER TABLE metrics
  -- Rename existing columns to indicate they track shared blocks
  RENAME COLUMN m_blk_read_time_cnt to m_shared_blk_read_time_cnt,
  -- m_shared_blk_read_time_sum: Total time the statement spent reading shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).
  RENAME COLUMN m_blk_read_time_sum to m_shared_blk_read_time_sum,
  RENAME COLUMN m_blk_write_time_cnt to m_shared_blk_write_time_cnt,
  -- m_shared_blk_write_time_sum: Total time the statement spent writing shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).
  RENAME COLUMN m_blk_write_time_sum to m_shared_blk_write_time_sum,
  -- Add new columns for local block I/O time tracking
  ADD COLUMN m_local_blk_read_time_cnt Float32,
  ADD COLUMN m_local_blk_read_time_sum Float32 COMMENT 'Total time the statement spent reading local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  ADD COLUMN m_local_blk_write_time_cnt Float32,
  ADD COLUMN m_local_blk_write_time_sum Float32 COMMENT 'Total time the statement spent writing local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).';
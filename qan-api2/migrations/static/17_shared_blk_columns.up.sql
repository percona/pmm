ALTER TABLE metrics
  RENAME COLUMN m_blk_read_time_cnt to m_shared_blk_read_time_cnt,
  RENAME COLUMN m_blk_read_time_sum to m_shared_blk_read_time_sum,
  COMMENT COLUMN m_shared_blk_read_time_sum 'Total time the statement spent reading shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  RENAME COLUMN m_blk_write_time_cnt to m_shared_blk_write_time_cnt,
  RENAME COLUMN m_blk_write_time_sum to m_shared_blk_write_time_sum,
  COMMENT COLUMN m_shared_blk_write_time_sum 'Total time the statement spent writing shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  ADD COLUMN m_local_blk_read_time_cnt Float32,
  ADD COLUMN m_local_blk_read_time_sum Float32 COMMENT 'Total time the statement spent reading local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  ADD COLUMN m_local_blk_write_time_cnt Float32,
  ADD COLUMN m_local_blk_write_time_sum Float32 COMMENT 'Total time the statement spent writing local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).';
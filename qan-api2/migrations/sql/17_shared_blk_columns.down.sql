ALTER TABLE metrics
  DROP COLUMN m_local_blk_read_time_cnt,
  DROP COLUMN m_local_blk_read_time_sum,
  DROP COLUMN m_local_blk_write_time_cnt,
  DROP COLUMN m_local_blk_write_time_sum,
  RENAME COLUMN m_shared_blk_read_time_cnt to  m_blk_read_time_cnt,
  RENAME COLUMN m_shared_blk_read_time_sum to m_blk_read_time_sum,
  COMMENT COLUMN m_blk_read_time_sum 'Total time the statement spent reading blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  RENAME COLUMN m_shared_blk_write_time_cnt to m_blk_write_time_cnt,
  RENAME COLUMN m_shared_blk_write_time_sum to m_blk_write_time_sum,
  COMMENT COLUMN m_blk_write_time_sum 'Total time the statement spent writing blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).';
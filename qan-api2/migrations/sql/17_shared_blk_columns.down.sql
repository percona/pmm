ALTER TABLE metrics
  DROP COLUMN m_local_blk_read_time_cnt,
  DROP COLUMN m_local_blk_read_time_sum,
  DROP COLUMN m_local_blk_write_time_cnt,
  DROP COLUMN m_local_blk_write_time_sum,
  RENAME COLUMN m_shared_blk_read_time_cnt to  m_blk_read_time_cnt,
  RENAME COLUMN m_shared_blk_read_time_sum to m_blk_read_time_sum,
  RENAME COLUMN m_shared_blk_write_time_cnt to m_blk_write_time_cnt,
  RENAME COLUMN m_shared_blk_write_time_sum to m_blk_write_time_sum;
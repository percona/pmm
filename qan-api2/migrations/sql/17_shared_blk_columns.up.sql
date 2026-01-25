ALTER TABLE metrics
  RENAME COLUMN m_blk_read_time_cnt to m_shared_blk_read_time_cnt,
  RENAME COLUMN m_blk_read_time_sum to m_shared_blk_read_time_sum,
  RENAME COLUMN m_blk_write_time_cnt to m_shared_blk_write_time_cnt,
  RENAME COLUMN m_blk_write_time_sum to m_shared_blk_write_time_sum,
  ADD COLUMN m_local_blk_read_time_cnt Float32,
  ADD COLUMN m_local_blk_read_time_sum Float32,
  ADD COLUMN m_local_blk_write_time_cnt Float32,
  ADD COLUMN m_local_blk_write_time_sum Float32;
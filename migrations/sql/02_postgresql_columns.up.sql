ALTER TABLE metrics
  ADD COLUMN `m_shared_blks_cnt` Float32,
  ADD COLUMN `m_shared_blks_hit` Float32 COMMENT 'Total number of shared block cache hits by the statement',
  ADD COLUMN `m_shared_blks_read` Float32 COMMENT 'Total number of shared blocks read by the statement.',
  ADD COLUMN `m_shared_blks_dirtied` Float32 COMMENT 'Total number of shared blocks dirtied by the statement.',
  ADD COLUMN `m_shared_blks_written` Float32 COMMENT 'Total number of shared blocks written by the statement.',
  ADD COLUMN `m_local_blks_cnt` Float32,
  ADD COLUMN `m_local_blks_hit` Float32 COMMENT 'Total number of local block cache hits by the statement',
  ADD COLUMN `m_local_blks_read` Float32 COMMENT 'Total number of local blocks read by the statement.',
  ADD COLUMN `m_local_blks_dirtied` Float32 COMMENT 'Total number of local blocks dirtied by the statement.',
  ADD COLUMN `m_local_blks_written` Float32 COMMENT 'Total number of local blocks written by the statement.',
  ADD COLUMN `m_temp_blks_cnt` Float32,
  ADD COLUMN `m_temp_blks_read` Float32 COMMENT 'Total number of temp blocks read by the statement.',
  ADD COLUMN `m_temp_blks_written` Float32 COMMENT 'Total number of temp blocks written by the statement.',
  ADD COLUMN `m_blk_read_time` Float32 COMMENT 'Total time the statement spent reading blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).',
  ADD COLUMN `m_blk_write_time` Float32 COMMENT 'Total time the statement spent writing blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).'
  ;
ALTER TABLE metrics
  ADD COLUMN `m_docs_examined_cnt` Float32,
  ADD COLUMN `m_docs_examined_sum` Float32,
  ADD COLUMN `m_docs_examined_min` Float32,
  ADD COLUMN `m_docs_examined_max` Float32,
  ADD COLUMN `m_docs_examined_p99` Float32,

  ADD COLUMN `m_keys_examined_cnt` Float32,
  ADD COLUMN `m_keys_examined_sum` Float32,
  ADD COLUMN `m_keys_examined_min` Float32,
  ADD COLUMN `m_keys_examined_max` Float32,
  ADD COLUMN `m_keys_examined_p99` Float32,

  ADD COLUMN `m_locks_global_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_global_acquire_count_read_shared_sum` Float32,

  ADD COLUMN `m_locks_global_acquire_count_write_shared_cnt` Float32,
  ADD COLUMN `m_locks_global_acquire_count_write_shared_sum` Float32,

  ADD COLUMN `m_locks_database_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_acquire_count_read_shared_sum` Float32,

  ADD COLUMN `m_locks_database_acquire_wait_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_acquire_wait_count_read_shared_sum` Float32,

  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_sum` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_min` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_max` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_p99` Float32,

  ADD COLUMN `m_locks_collection_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_collection_acquire_count_read_shared_sum` Float32,

  ADD COLUMN `m_storage_bytes_read_cnt` Float32,
  ADD COLUMN `m_storage_bytes_read_sum` Float32,
  ADD COLUMN `m_storage_bytes_read_min` Float32,
  ADD COLUMN `m_storage_bytes_read_max` Float32,
  ADD COLUMN `m_storage_bytes_read_p99` Float32,

  ADD COLUMN `m_storage_time_reading_micros_cnt` Float32,
  ADD COLUMN `m_storage_time_reading_micros_sum` Float32,
  ADD COLUMN `m_storage_time_reading_micros_min` Float32,
  ADD COLUMN `m_storage_time_reading_micros_max` Float32,
  ADD COLUMN `m_storage_time_reading_micros_p99` Float32;

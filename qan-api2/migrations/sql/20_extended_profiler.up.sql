ALTER TABLE metrics
  ADD COLUMN `m_docs_examined_cnt` Float32,
  ADD COLUMN `m_docs_examined_sum` Float32 
  COMMENT 'Total number of documents scanned during query execution',
  ADD COLUMN `m_docs_examined_min` Float32,
  ADD COLUMN `m_docs_examined_max` Float32,
  ADD COLUMN `m_docs_examined_p99` Float32,

  ADD COLUMN `m_keys_examined_cnt` Float32,
  ADD COLUMN `m_keys_examined_sum` Float32
  COMMENT 'Total number of index keys scanned during query execution',
  ADD COLUMN `m_keys_examined_min` Float32,
  ADD COLUMN `m_keys_examined_max` Float32,
  ADD COLUMN `m_keys_examined_p99` Float32,

  ADD COLUMN `m_locks_global_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_global_acquire_count_read_shared_sum` Float32 
  COMMENT 'Number of times a global read lock was acquired during query execution',

  ADD COLUMN `m_locks_global_acquire_count_write_shared_cnt` Float32,
  ADD COLUMN `m_locks_global_acquire_count_write_shared_sum` Float32
  COMMENT 'Number of times a global write lock was acquired during query execution',

  ADD COLUMN `m_locks_database_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_acquire_count_read_shared_sum` Float32
  COMMENT 'Number of times a read lock was acquired at the database level during query execution',

  ADD COLUMN `m_locks_database_acquire_wait_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_acquire_wait_count_read_shared_sum` Float32
  COMMENT 'Number of times a read lock at the database level was requested but had to wait before being granted',

  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_sum` Float32 
  COMMENT 'Indicates the time, spent acquiring a read lock at the database level during an operation',
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_min` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_max` Float32,
  ADD COLUMN `m_locks_database_time_acquiring_micros_read_shared_p99` Float32,

  ADD COLUMN `m_locks_collection_acquire_count_read_shared_cnt` Float32,
  ADD COLUMN `m_locks_collection_acquire_count_read_shared_sum` Float32
  COMMENT 'Number of times a read lock was acquired on a specific collection during operations',

  ADD COLUMN `m_storage_bytes_read_cnt` Float32,
  ADD COLUMN `m_storage_bytes_read_sum` Float32 
  COMMENT 'Total number of bytes read from storage during a specific operation',
  ADD COLUMN `m_storage_bytes_read_min` Float32,
  ADD COLUMN `m_storage_bytes_read_max` Float32,
  ADD COLUMN `m_storage_bytes_read_p99` Float32,

  ADD COLUMN `m_storage_time_reading_micros_cnt` Float32,
  ADD COLUMN `m_storage_time_reading_micros_sum` Float32 
  COMMENT 'Indicates the time, spent reading data from storage during an operation',
  ADD COLUMN `m_storage_time_reading_micros_min` Float32,
  ADD COLUMN `m_storage_time_reading_micros_max` Float32,
  ADD COLUMN `m_storage_time_reading_micros_p99` Float32;

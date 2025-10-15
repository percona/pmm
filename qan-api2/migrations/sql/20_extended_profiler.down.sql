ALTER TABLE metrics
  DROP COLUMN `m_docs_examined_cnt`,
  DROP COLUMN `m_docs_examined_sum`,
  DROP COLUMN `m_docs_examined_min`,
  DROP COLUMN `m_docs_examined_max`,
  DROP COLUMN `m_docs_examined_p99`,

  DROP COLUMN `m_keys_examined_cnt`,
  DROP COLUMN `m_keys_examined_sum`,
  DROP COLUMN `m_keys_examined_min`,
  DROP COLUMN `m_keys_examined_max`,
  DROP COLUMN `m_keys_examined_p99`,

  DROP COLUMN `m_locks_global_acquire_count_read_shared_cnt`,
  DROP COLUMN `m_locks_global_acquire_count_read_shared_sum`,

  DROP COLUMN `m_locks_global_acquire_count_write_shared_cnt`,
  DROP COLUMN `m_locks_global_acquire_count_write_shared_sum`,

  DROP COLUMN `m_locks_database_acquire_count_read_shared_cnt`,
  DROP COLUMN `m_locks_database_acquire_count_read_shared_sum`,

  DROP COLUMN `m_locks_database_acquire_wait_count_read_shared_cnt`,
  DROP COLUMN `m_locks_database_acquire_wait_count_read_shared_sum`,

  DROP COLUMN `m_locks_database_time_acquiring_micros_read_shared_cnt`,
  DROP COLUMN `m_locks_database_time_acquiring_micros_read_shared_sum`,
  DROP COLUMN `m_locks_database_time_acquiring_micros_read_shared_min`,
  DROP COLUMN `m_locks_database_time_acquiring_micros_read_shared_max`,
  DROP COLUMN `m_locks_database_time_acquiring_micros_read_shared_p99`,

  DROP COLUMN `m_locks_collection_acquire_count_read_shared_cnt`,
  DROP COLUMN `m_locks_collection_acquire_count_read_shared_sum`,

  DROP COLUMN `m_storage_bytes_read_cnt`,
  DROP COLUMN `m_storage_bytes_read_sum`,
  DROP COLUMN `m_storage_bytes_read_min`,
  DROP COLUMN `m_storage_bytes_read_max`,
  DROP COLUMN `m_storage_bytes_read_p99`,

  DROP COLUMN `m_storage_time_reading_micros_cnt`,
  DROP COLUMN `m_storage_time_reading_micros_sum`,
  DROP COLUMN `m_storage_time_reading_micros_min`,
  DROP COLUMN `m_storage_time_reading_micros_max`,
  DROP COLUMN `m_storage_time_reading_micros_p99`;

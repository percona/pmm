DROP INDEX IF EXISTS idx_load_test_status;
ALTER TABLE load_test DROP COLUMN status;
DROP TYPE status_enum; 
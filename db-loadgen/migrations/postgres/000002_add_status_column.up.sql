CREATE TYPE status_enum AS ENUM ('active', 'inactive', 'pending');
ALTER TABLE load_test ADD COLUMN status status_enum DEFAULT 'active';
CREATE INDEX IF NOT EXISTS idx_load_test_status ON load_test (status); 
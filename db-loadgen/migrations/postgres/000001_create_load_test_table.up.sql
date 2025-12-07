CREATE TABLE IF NOT EXISTS load_test (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_load_test_name ON load_test (name);
CREATE INDEX IF NOT EXISTS idx_load_test_value ON load_test (value);
CREATE INDEX IF NOT EXISTS idx_load_test_created_at ON load_test (created_at);

-- Function to update updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_load_test_updated_at 
    BEFORE UPDATE ON load_test 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column(); 
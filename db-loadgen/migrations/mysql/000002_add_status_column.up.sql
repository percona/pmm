ALTER TABLE load_test ADD COLUMN status ENUM('active', 'inactive', 'pending') DEFAULT 'active';

CREATE INDEX idx_status ON load_test (status); 
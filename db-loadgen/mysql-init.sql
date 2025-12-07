-- MySQL initialization script for PMM monitoring
-- This script creates users with necessary privileges for both load generation and PMM monitoring

-- Create a PMM monitoring user with full privileges for performance schema access
CREATE USER IF NOT EXISTS 'pmm-mysql'@'%' IDENTIFIED BY 'pmm-pass';

-- Grant comprehensive privileges needed for PMM monitoring
GRANT SELECT, PROCESS, SUPER, REPLICATION CLIENT, RELOAD ON *.* TO 'pmm-mysql'@'%';

-- Grant specific privileges for performance schema tables
GRANT SELECT ON performance_schema.* TO 'pmm-mysql'@'%';

-- Grant privileges for query analytics
GRANT SELECT ON mysql.* TO 'pmm-mysql'@'%';

-- Grant privileges for the test database
GRANT ALL PRIVILEGES ON testdb.* TO 'pmm-mysql'@'%';

-- Also grant performance schema access to the existing testuser for compatibility
GRANT SELECT ON performance_schema.* TO 'testuser'@'%';
GRANT PROCESS ON *.* TO 'testuser'@'%';

-- Flush privileges to ensure they take effect
FLUSH PRIVILEGES;

-- Show created users for verification
SELECT User, Host FROM mysql.user WHERE User IN ('testuser', 'pmm-mysql'); 
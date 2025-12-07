-- PostgreSQL initialization script for PMM monitoring
-- This script enables necessary extensions and creates users for PMM monitoring

-- Enable pg_stat_statements extension for query statistics
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Enable pg_stat_monitor extension for enhanced query monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_monitor;

-- Create a PMM monitoring user with necessary privileges
CREATE USER "pmm-postgres" WITH PASSWORD 'pmm-pass';

-- Grant necessary privileges for PMM monitoring
GRANT pg_monitor TO "pmm-postgres";

-- Grant privileges to read system catalogs and statistics
GRANT SELECT ON ALL TABLES IN SCHEMA information_schema TO "pmm-postgres";
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO "pmm-postgres";

-- Grant privileges for the test database
GRANT ALL PRIVILEGES ON DATABASE testdb TO "pmm-postgres";

-- Connect to testdb to grant schema privileges
\c testdb

-- Grant privileges on the testdb schemas
GRANT USAGE ON SCHEMA public TO "pmm-postgres";
GRANT SELECT ON ALL TABLES IN SCHEMA public TO "pmm-postgres";
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "pmm-postgres";

-- Grant default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO "pmm-postgres";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON SEQUENCES TO "pmm-postgres";

-- Enable pg_stat_statements and pg_stat_monitor extensions in testdb as well
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS pg_stat_monitor;

-- Show enabled extensions for verification
SELECT extname, extversion FROM pg_extension WHERE extname IN ('pg_stat_statements', 'pg_stat_monitor'); 
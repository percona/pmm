-- Clean break: drop the legacy qan-api2 wide table. The new schema is incompatible
-- and pre-upgrade history is not migrated (users are warned in the release notes).
DROP TABLE IF EXISTS metrics;

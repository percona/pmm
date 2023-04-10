CREATE DATABASE "pmm-managed";
CREATE USER "pmm-managed" WITH PASSWORD 'pmm-managed';
GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO "pmm-managed";
\c pmm-managed;
CREATE EXTENSION pg_stat_statements SCHEMA public;

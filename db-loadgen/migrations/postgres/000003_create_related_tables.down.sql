DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_users_updated_at_column();
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS operation_type; 
-- Simple template file for testing
CREATE TABLE {{.DatabaseName}}.{{.TableName}} (
  id BIGINT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

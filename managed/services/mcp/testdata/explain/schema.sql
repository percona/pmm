DROP DATABASE IF EXISTS shop;
CREATE DATABASE shop;
USE shop;
CREATE TABLE customers (
  id INT PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  region VARCHAR(16) NOT NULL,
  KEY idx_region (region)
) ENGINE=InnoDB;
CREATE TABLE orders (
  id INT PRIMARY KEY,
  customer_id INT NOT NULL,
  total DECIMAL(10,2) NOT NULL,
  status VARCHAR(16) NOT NULL,
  KEY idx_customer (customer_id)
) ENGINE=InnoDB;
CREATE TABLE a (
  id INT PRIMARY KEY,
  b_id INT NOT NULL,
  v INT NOT NULL
) ENGINE=InnoDB;
CREATE TABLE b (
  id INT PRIMARY KEY,
  a_id INT NOT NULL,
  v INT NOT NULL
) ENGINE=InnoDB;
INSERT INTO customers VALUES (1,'alice','eu'),(2,'bob','us'),(3,'carol','eu');
INSERT INTO orders VALUES (1,1,10.00,'new'),(2,1,20.00,'paid'),(3,2,30.00,'new'),(4,3,5.00,'paid');
INSERT INTO a VALUES (1,1,1),(2,2,2),(3,3,3);
INSERT INTO b VALUES (1,1,1),(2,2,2),(3,3,3);
ANALYZE TABLE customers, orders, a, b;

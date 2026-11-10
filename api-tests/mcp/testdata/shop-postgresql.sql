DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS customers;
CREATE TABLE customers (
  id integer PRIMARY KEY,
  name varchar(64) NOT NULL,
  region varchar(16) NOT NULL
);
CREATE INDEX idx_region ON customers (region);
CREATE TABLE orders (
  id integer PRIMARY KEY,
  customer_id integer NOT NULL,
  total numeric(10,2) NOT NULL,
  status varchar(16) NOT NULL
);
CREATE INDEX idx_customer ON orders (customer_id);
INSERT INTO customers VALUES (1,'alice','eu'),(2,'bob','us'),(3,'carol','eu');
INSERT INTO orders VALUES (1,1,10.00,'new'),(2,1,20.00,'paid'),(3,2,30.00,'new'),(4,3,5.00,'paid');
ANALYZE customers, orders;

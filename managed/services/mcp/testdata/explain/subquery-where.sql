SELECT id FROM customers WHERE id IN (SELECT customer_id FROM orders WHERE total > 10)

SELECT id FROM orders WHERE status = 'new' UNION SELECT id FROM customers WHERE region = 'eu'

SELECT o.id, c.name FROM orders o JOIN customers c ON c.id = o.customer_id WHERE c.region = 'eu'

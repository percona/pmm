SELECT orders.id, customers.name FROM orders JOIN customers ON customers.id = orders.customer_id WHERE customers.region = 'eu'

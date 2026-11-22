SELECT CASE WHEN o.status = 'zq-paid-8842' THEN 'zq-done-1193' ELSE 'zq-open-2284' END, IFNULL(c.name, 'zq-anon-3375') FROM orders o JOIN customers c ON c.id = o.customer_id

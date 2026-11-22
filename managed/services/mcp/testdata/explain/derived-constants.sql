SELECT * FROM (SELECT customer_id, 'zq-tier-4401', SUM(total) * 2.0517 FROM orders GROUP BY customer_id) d JOIN customers c ON c.id = d.customer_id

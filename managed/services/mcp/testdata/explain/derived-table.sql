SELECT d.customer_id, d.s FROM (SELECT customer_id, SUM(total) AS s FROM orders GROUP BY customer_id) d JOIN customers c ON c.id = d.customer_id

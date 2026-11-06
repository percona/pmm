WITH t AS (SELECT customer_id, SUM(total) AS s FROM orders GROUP BY customer_id) SELECT c.name, t.s FROM t JOIN customers c ON c.id = t.customer_id

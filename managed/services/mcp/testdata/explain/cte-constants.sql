WITH t AS (SELECT customer_id, 'zq-cte-5512', SUM(total) * 3.0419 FROM orders GROUP BY customer_id) SELECT * FROM t JOIN customers c ON c.id = t.customer_id

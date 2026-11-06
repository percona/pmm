WITH orders AS (SELECT id FROM customers) SELECT o.id FROM orders o JOIN shop.orders x ON x.customer_id = o.id

SELECT first_name, last_name, email
FROM customer
WHERE customer_id IN (
SELECT customer_id FROM (
SELECT DISTINCT customer_id, SUM(amount)
FROM payment
WHERE extract(month from payment_date) = 4
AND extract(day from payment_date) BETWEEN 10 AND 13
GROUP BY customer_id
HAVING SUM(amount) > 30
ORDER BY SUM(amount) DESC
LIMIT 5) AS top_five);

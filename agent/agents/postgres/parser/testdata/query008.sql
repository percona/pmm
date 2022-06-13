SELECT date_part('day', p.payment_date)::INT AS legit,
SUM(p.amount),
date_part('day', fk.setup)::INT AS fake
FROM payment AS p
LEFT JOIN fake_month AS fk
ON date_part('day', fk.setup)::INT = date_part('day', p.payment_date)::INT
GROUP BY legit, fake
HAVING SUM(p.amount) > $1
ORDER BY fake NULLS first
LIMIT $2;

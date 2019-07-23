WITH fake_month AS(
SELECT setup::date
FROM generate_series('2007-02-01', '2007-02-28', INTERVAL '1 day') AS setup
)
SELECT date_part('day', p.payment_date)::INT AS legit,
SUM(p.amount),
date_part('day', fk.setup)::INT AS fake
FROM payment AS p
LEFT JOIN fake_month AS fk
ON date_part('day', fk.setup)::INT = date_part('day', p.payment_date)::INT
GROUP BY legit, fake
HAVING SUM(p.amount) > $1
LIMIT $2;

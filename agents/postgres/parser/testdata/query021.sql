INSERT INTO sales.big_orders (id, full_name, address, total)
SELECT
   id,
   full_name,
   address,
   total
FROM
   sales.total_orders
WHERE
   total > $1;


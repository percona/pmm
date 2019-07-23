 UPDATE employees SET sales_count = sales_count + 1 WHERE id =
   (SELECT sales_person FROM accounts WHERE name = 'Acme Corporation');

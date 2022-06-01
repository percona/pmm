select * from city c inner join country c2 on c.countrycode = c2.code 
where countrycode = (SELECT c3.countrycode from countrylanguage c3 where c3.countrycode = 'KGZ' limit 1) 
limit 90

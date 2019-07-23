select n1.name, n1.author_id, count_1, total_count
from (select id, name, author_id, count(1) as count_1
      from names
      group by id, name, author_id) n1
inner join (select id, author_id, count(1) as total_count
          from names
          group by id, author_id) n2
on (n2.id = n1.id and n2.author_id = n1.author_id)

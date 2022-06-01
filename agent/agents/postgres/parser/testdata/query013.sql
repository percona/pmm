select d.*
from order_line_detail d
where d.line_id in (
  select l.id
  from order_line l
  where l.order_id in (
      select o.id
      from orders o
      where o.last_update > now() - interval '6 hours'
  )
);

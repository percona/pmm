with cte_order as (
    select o.id 
    from orders o
    where o.last_update > now() - interval '6 hours'
),
cte_line as (
    select l.id
    from order_line l
    where l.order_id in (
        select * from cte_order
    )
)
select d.*
from order_line_details d
where d.id in (select * from cte_line)

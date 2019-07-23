with order_price_transport as (
    select orders.id, price_parts.code, price_parts.status, sum(price_parts.amount)
    from orders
    join price_parts on price_parts.order_id = orders.id
    where price_parts.code in ('FUEL', FREIGHT)
    group by orders.id, price_parts.code, price_parts.status
),
order_price_other as (
    select orders.id, price_parts.code, price_parts.status, sum(price_parts.amount)
    from orders
    join price_parts on price_parts.order_id = orders.id
    where price_parts.code not in ('FUEL', 'FREIGHT')
    group by orders.id, price_parts.code, price_parts.status
)
select
    orders.id,
    est_trans_price.amount,
    act_trans_price.amount,
    est_other_price.amount,
    act_other_price.amount
from orders
join order_price_transport as est_trans_price on est_trans_price.id = orders.id and est_trans_price.status = 'estimated'
join order_price_transport as act_trans_price on act_trans_price.id = orders.id and act_trans_price.status = 'actual'
join order_price_other as est_other_price on est_other_price.id = orders.id and est_other_price.status = 'estimated'
join order_price_other as act_other_price on act_trans_price.id = orders.id and act_other_price.status = 'actual'
where orders.creation_date between $1 and $2
and orders.organization_id in ($3)
and orders.status_id in (123, 456)


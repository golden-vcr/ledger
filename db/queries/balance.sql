-- name: GetBalance :one
select
    balance.total_points::integer as total_points,
    balance.available_points::integer as available_points
from ledger.balance
where twitch_user_id = @twitch_user_id;

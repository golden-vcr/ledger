-- name: GetBalance :one
select
    *
from ledger.balance
where twitch_user_id = @twitch_user_id;

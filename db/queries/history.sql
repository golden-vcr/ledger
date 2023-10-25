-- name: GetTransactionHistory :many
select
    flow.id,
    flow.type,
    flow.metadata,
    flow.delta_points,
    flow.created_at,
    flow.finalized_at,
    flow.accepted
from ledger.flow
where flow.twitch_user_id = @twitch_user_id
and case when sqlc.narg('start_id')::uuid is null
    then true
    else flow.created_at <= (
        select flow.created_at from ledger.flow where flow.id = sqlc.narg('start_id')::uuid
    )
end
order by flow.created_at desc
limit @num_records;

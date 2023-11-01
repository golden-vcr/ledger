-- name: GetFlow :one
select
    twitch_user_id,
    finalized_at,
    accepted
from ledger.flow
where flow.id = @flow_id;

-- name: FinalizeFlow :execresult
update ledger.flow set
    finalized_at = now(),
    accepted = @accepted
where
    flow.id = @flow_id
    and finalized_at is null;

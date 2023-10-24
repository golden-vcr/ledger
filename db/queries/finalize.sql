-- name: FinalizeFlow :execresult
update ledger.flow set
    finalized_at = now(),
    accepted = @accepted
where
    flow.id = @flow_id
    and finalized_at is null;

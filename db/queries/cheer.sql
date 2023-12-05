-- name: RecordCheerInflow :one
insert into ledger.flow (
    id,
    type,
    metadata,
    twitch_user_id,
    delta_points,
    created_at,
    finalized_at,
    accepted
) values (
    gen_random_uuid(),
    'cheer',
    jsonb_build_object('message', @message::text),
    @twitch_user_id,
    @num_points_to_credit,
    now(),
    now(),
    true
)
returning flow.id;

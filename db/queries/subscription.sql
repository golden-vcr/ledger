-- name: RecordSubscriptionInflow :one
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
    'subscription',
    jsonb_build_object(
        'message', @message::text,
        'is_initial', @is_initial::boolean,
        'is_gift', @is_gift::boolean,
        'credit_multiplier', @credit_multiplier::float
    ),
    @twitch_user_id,
    @num_points_to_credit,
    now(),
    now(),
    true
)
returning flow.id;

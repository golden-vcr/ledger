-- name: RecordGiftSubInflow :one
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
    'gift-sub',
    jsonb_build_object(
        'num_subscriptions', @num_subscriptions::integer,
        'credit_multiplier', @credit_multiplier::float
    ),
    @twitch_user_id,
    @num_points_to_credit,
    now(),
    now(),
    true
)
returning flow.id;

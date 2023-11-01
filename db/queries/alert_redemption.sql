-- name: RecordPendingAlertRedemptionOutflow :one
insert into ledger.flow (
    id,
    type,
    metadata,
    twitch_user_id,
    delta_points,
    created_at
) values (
    gen_random_uuid(),
    'alert-redemption',
    (case when sqlc.narg('alert_metadata')::jsonb is null
        then '{}'::jsonb
        else sqlc.narg('alert_metadata')::jsonb
    end) || jsonb_build_object('type', @alert_type::text),
    @twitch_user_id,
    -1 * @num_points_to_debit::integer,
    now()
)
returning flow.id;

-- name: StoreSseToken :exec
insert into ledger.sse_token (
    twitch_user_id,
    value,
    expires_at
) values (
    @twitch_user_id,
    @token_value,
    now() + ((@ttl_seconds::int)::text || 's')::interval
);

-- name: PurgeSseTokensForUser :exec
delete from ledger.sse_token
where sse_token.twitch_user_id = @twitch_user_id
    and sse_token.expires_at <= now();

-- name: IdentifyUserFromSseToken :one
select
    sse_token.twitch_user_id
from ledger.sse_token
where sse_token.value = @token_value
    and sse_token.expires_at > now()
limit 1;

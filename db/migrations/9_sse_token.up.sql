begin;

create table ledger.sse_token (
    twitch_user_id text not null,
    value          text not null,
    expires_at     timestamptz not null
);

comment on table ledger.sse_token is
    'Record of a short-lived cryptographic token used to authenticate the given user, '
    'solely for the purpose of allowing them access to real-time transaction data via '
    'the /notifications SSE endpoint.';
comment on column ledger.sse_token.twitch_user_id is
    'ID of the user whose transaction notifications should be sent to the bearer of '
    'this token.';
comment on column ledger.sse_token.value is
    'String token value, typically a hex-encoded cryptographically random string.';
comment on column ledger.sse_token.expires_at is
    'Time at which the token should no longer be accepted (and may be purged).';

commit;

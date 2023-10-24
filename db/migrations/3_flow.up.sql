begin;

create table ledger.flow (
    id             uuid primary key,
    type           text not null references ledger.flow_type (name) on update cascade,
    metadata       jsonb not null default '{}'::jsonb,
    twitch_user_id text not null,
    delta_points   integer not null,
    created_at     timestamptz not null default now(),
    finalized_at   timestamptz,
    accepted       boolean not null default false
);

comment on table ledger.flow is
    'Record of a single transaction, i.e. an inflow or an outflow, that credits points '
    'to or debits points from a given user. A transaction may initially exist in a '
    'pending state, in which case the finalized_at timestamp will be null. A pending '
    'transaction will eventually be finalized, at which point it is either accepted or '
    'rejected. A pending inflow counts toward the user''s total balance but does not '
    'contribute to their available balance until accepted. A pending outflow '
    'immediately deducts from the user''s available balance, but does not reduce their '
    'total balance until accepted. Any transaction that''s rejected will be retained '
    'for record-keeping purposes but will have no effect on any balances.';
comment on column ledger.flow.id is
    'Unique ID to serve as a handle for this ledger transaction.';
comment on column ledger.flow.type is
    'Type of transaction, corresponding to a valid name from the flow_type table. The '
    'corresponding flow_type.is_inflow value indicates whether this transaction is an '
    'inflow or an outflow.';
comment on column ledger.flow.metadata is
    'Additional JSON metadata providing context about this transaction. The exact '
    'semantics of this object are flow-type-dependent; each valid flow_type should '
    'describe the required/supported metadata fields and impose constraints on this '
    'column where necessary.';
comment on column ledger.flow.twitch_user_id is
    'ID of the user whose balance this transaction affects.';
comment on column ledger.flow.delta_points is
    'Change in point balance to be applied by this transaction: positive for an '
    'inflow; negative for an outflow.';
comment on column ledger.flow.created_at is
    'Time at which this transaction was initially recorded.';
comment on column ledger.flow.finalized_at is
    'Time at which this transaction was finalized. If NULL, the transaction is '
    'pending.';
comment on column ledger.flow.accepted is
    'For a finalized transaction, indicates whether the transaction was accepted or '
    'rejected.';

alter table ledger.flow
    add constraint flow_delta_points_check
    check (
        delta_points != 0
    );

comment on constraint flow_delta_points_check on ledger.flow is
    'Ensures that delta_points will never be 0.';

alter table ledger.flow
    add constraint flow_accepted_finalized_at_check
    check (
        case when flow.accepted
            then flow.finalized_at is not null
            else true
        end
    );

comment on constraint flow_accepted_finalized_at_check on ledger.flow is
    'Ensures that a flow may only be marked as accepted when it has a valid '
    'finalized_at timestamp.';

commit;

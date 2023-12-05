begin;

alter table ledger.flow
    add column affects_total_balance boolean generated always as (
        case when finalized_at is null
            -- A pending inflow should add to the total balance regardless of whether
            -- it's available yet; a pending outflow should not deduct from the total
            -- balance until accepted
            then delta_points > 0
            -- Any finalized transaction should count toward all balances if accepted,
            -- and should be ignored if rejected
            else accepted
        end
    )
    stored;

comment on column ledger.flow.affects_total_balance is
    'Whether this transaction should affect the user''s total point balance, computed '
    'as a function of delta_points, finalized_at, and accepted.';

alter table ledger.flow
    add column affects_available_balance boolean generated always as (
        case when finalized_at is null
            -- A pending inflow should not be available; a pending outflow should deduct
            -- from the available balance until finalized
            then delta_points < 0
            -- Any finalized transaction should count toward all balances if accepted,
            -- and should be ignored if rejected
            else accepted
        end
    )
    stored;

comment on column ledger.flow.affects_available_balance is
    'Whether this transaction should affect the user''s available point balance, '
    'computed as a function of delta_points, finalized_at, and accepted.';

commit;

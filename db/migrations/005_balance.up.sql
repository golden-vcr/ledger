begin;

create view ledger.balance as
    select
        flow.twitch_user_id,
        coalesce(
            sum(flow.delta_points) filter (where flow.affects_total_balance),
            0
         ) as total_points,
        coalesce(
            sum(flow.delta_points) filter (where flow.affects_available_balance),
            0
         ) as available_points
    from ledger.flow
    group by flow.twitch_user_id;

comment on view ledger.balance is
    'Lookup describing the total and available point balance for each user, based on '
    'the aggregate of all inflows and outflows recorded for that user.';
comment on column ledger.balance.twitch_user_id is
    'ID of the user for whom we''re summarizing transaction data.';
comment on column ledger.balance.total_points is
    'Total number of points credited to this user currently.';
comment on column ledger.balance.available_points is
    'Total number of points available for this user to spend.';

commit;

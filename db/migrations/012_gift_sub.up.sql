begin;

insert into ledger.flow_type (name, comment) values (
    'gift-sub',
    'Inflow triggered in response to a gift subscription event that occurs on Twitch; '
    'i.e. when the associated user has granted one or more gift subs to other users. '
    'This transaction credits the purchaser of the gift sub(s); recipients of those '
    'subs are credited via the ordinary ''subscription'' inflow. The required metadata '
    'fields ''num_subscriptions'' (int) and ''credit_multiplier'' (float) indicate how '
    'many subs were gifted and the value multiplier of the selected subscription tier, '
    'respectively.'
);

alter table ledger.flow
    add constraint flow_gift_sub_check check (
        case when flow.type != 'gift-sub' then true else
            flow.delta_points > 0
            and jsonb_typeof(flow.metadata->'num_subscriptions') = 'number'
            and jsonb_typeof(flow.metadata->'credit_multiplier') = 'number'
        end
    );

comment on constraint flow_gift_sub_check on ledger.flow is
    'Ensures that any transaction representing a gift sub purchase is an inflow and '
    'has both ''num_subscriptions'' and ''credit_multiplier'' metadata fields.';

commit;

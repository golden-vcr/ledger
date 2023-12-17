begin;

insert into ledger.flow_type (name, comment) values (
    'subscription',
    'Inflow triggered in response to a subscription event that occurs on Twitch; '
    'either a user purchasing an initial subscription, receiving a gift subscription, '
    'or sending a resub message to announce renewal of their existing subscription. '
    'The metadata.message field may store the message that accompanied a resub, if '
    'applicable, optionally truncated. The is_initial (bool), is_gift (bool), and '
    'credit_multiplier (float) fields must be specified.'
);

alter table ledger.flow
    add constraint flow_subscription_check check (
        case when flow.type != 'subscription' then true else
            flow.delta_points > 0
            and jsonb_typeof(flow.metadata->'message') = 'string'
            and jsonb_typeof(flow.metadata->'is_initial') = 'boolean'
            and jsonb_typeof(flow.metadata->'is_gift') = 'boolean'
            and jsonb_typeof(flow.metadata->'credit_multiplier') = 'number'
        end
    );

comment on constraint flow_subscription_check on ledger.flow is
    'Ensures that any transaction representing a subscription is an inflow and has '
    '''message'', ''is_initial'', ''is_gift'', and ''credit_multiplier'' metadata '
    'fields of the appropriate types.';

commit;

begin;

insert into ledger.flow_type (name, comment) values (
    'cheer',
    'Inflow triggered in response to a channel.cheer webhook notification, in order to '
    'grant a user points when they cheer with bits. The inflow''s metadata.message '
    'field may store the message that accompanied the cheer, optionally truncated.'
);

alter table ledger.flow
    add constraint flow_cheer_check check (
        case when flow.type != 'cheer' then true else
            flow.delta_points > 0
            and jsonb_typeof(flow.metadata->'message') = 'string'
        end
    );

comment on constraint flow_cheer_check on ledger.flow is
    'Ensures that any transaction representing a cheer is an inflow and has a '
    '''message'' field which may or may not be empty.';

commit;

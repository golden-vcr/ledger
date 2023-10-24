begin;

insert into ledger.flow_type (name, comment) values (
    'alert-redemption',
    'Outflow triggered when a user decides to redeem points in order to trigger an '
    'alert. The outflow''s metadata.type field must be set to a non-empty string '
    'indicating the type of alert that was requested. Other alert-type-specific '
    'metadata may be specified as needed; the ledger service makes no attempt to '
    'validate alert redemption metadata beyond ensuring that it has a type name.'
);

alter table ledger.flow
    add constraint flow_alert_redemption_check check (
        case when flow.type != 'alert-redemption' then true else
            flow.delta_points < 0
            and jsonb_typeof(flow.metadata->'type') = 'string'
            and flow.metadata->>'type' != ''
        end
    );

comment on constraint flow_alert_redemption_check on ledger.flow is
    'Ensures that any transaction representing an alert redemption is an outflow and '
    'has a valid ''type'' field recorded in its metadata.';

commit;

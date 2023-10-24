begin;

insert into ledger.flow_type (name, comment) values (
    'manual-credit',
    'Inflow triggered manually by an admin, in order to grant the user an arbitrary '
    'number of points at the admin''s discretion. The inflow''s metadata.note field '
    'must be set to a non-empty string describing the purpose of the credit.'
);

alter table ledger.flow
    add constraint flow_manual_credit_check check (
        case when flow.type != 'manual-credit' then true else
            flow.delta_points > 0
            and jsonb_typeof(flow.metadata->'note') = 'string'
            and flow.metadata->>'note' != ''
        end
    );

comment on constraint flow_manual_credit_check on ledger.flow is
    'Ensures that any transaction representing a manual credit is an inflow and has a '
    'valid ''note'' field recorded in its metadata.';

commit;

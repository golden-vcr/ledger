begin;

create table ledger.flow_type (
    name      text primary key,
    comment   text not null
);

comment on table ledger.flow_type is
    'Internal record of a valid type of flow (i.e. inflow or outflow) by which points '
    'can be credited to or debited from a user.';
comment on column ledger.flow_type.name is
    'Unique string name by which the flow type is identified, canonically kebab-case.';
comment on column ledger.flow_type.comment is
    'Developer-facing description of this flow type; including its purpose and a '
    'description of any additional metadata required for transactions of this type.';

commit;

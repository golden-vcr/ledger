begin;

alter table ledger.flow
    drop constraint flow_manual_credit_check;

delete from ledger.flow_type where name = 'manual-credit';

commit;

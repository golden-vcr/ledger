begin;

alter table ledger.flow
    drop column affects_total_balance;

alter table ledger.flow
    drop column affects_available_balance;

commit;

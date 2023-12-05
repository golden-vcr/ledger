begin;

alter table ledger.flow
    drop constraint flow_cheer_check;

delete from ledger.flow_type where name = 'cheer';

commit;

begin;

alter table ledger.flow
    drop constraint flow_gift_sub_check;

delete from ledger.flow_type where name = 'gift-sub';

commit;

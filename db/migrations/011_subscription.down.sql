begin;

alter table ledger.flow
    drop constraint flow_subscription_check;

delete from ledger.flow_type where name = 'subscription';

commit;

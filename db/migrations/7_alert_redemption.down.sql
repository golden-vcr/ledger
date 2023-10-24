begin;

alter table ledger.flow
    drop constraint flow_alert_redemption_check;

delete from ledger.flow_type where name = 'alert-redemption';

commit;

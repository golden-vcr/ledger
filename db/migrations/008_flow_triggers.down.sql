begin;

drop trigger notify_on_flow_change on ledger.flow;

drop function emit_flow_change_notification;

commit;

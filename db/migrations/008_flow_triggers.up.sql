begin;

create function emit_flow_change_notification() returns trigger as $trigger$
begin
    perform pg_notify('ledger_flow_change', jsonb_build_object(
	    'twitch_user_id', NEW.twitch_user_id,
	    'id', NEW.id,
	    'type', NEW.type,
	    'metadata', NEW.metadata,
	    'delta_points', NEW.delta_points,
	    'created_at', NEW.created_at,
	    'finalized_at', NEW.finalized_at,
	    'accepted', NEW.accepted
    )::text);
    return NEW;
end;
$trigger$ language plpgsql;

create trigger notify_on_flow_change
    after insert or update on ledger.flow
    for each row execute procedure emit_flow_change_notification();

commit;

CREATE OR REPLACE FUNCTION audit_events_prevent_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('openlicensd.audit_prune', true) IS DISTINCT FROM 'on' THEN
        RAISE EXCEPTION 'audit_events is append-only';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
CREATE TRIGGER audit_events_no_delete
    BEFORE DELETE ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION audit_events_prevent_delete();

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    resource_label TEXT,
    actor_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    actor_token_id UUID REFERENCES api_tokens (id) ON DELETE SET NULL,
    actor_name TEXT NOT NULL,
    actor_email TEXT,
    actor_token_prefix TEXT,
    actor_role TEXT NOT NULL,
    auth_method TEXT NOT NULL,
    client_ip TEXT,
    user_agent TEXT,
    request_id TEXT,
    request_method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    response_status INTEGER NOT NULL,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at ON audit_events (occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor_user_id ON audit_events (actor_user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events (action, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events (resource_type, resource_id);

ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_auth_method_check;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_auth_method_check
    CHECK (auth_method IN ('session', 'api_token'));

CREATE OR REPLACE FUNCTION audit_events_prevent_update()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
CREATE TRIGGER audit_events_no_update
    BEFORE UPDATE ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION audit_events_prevent_update();

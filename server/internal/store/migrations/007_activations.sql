ALTER TABLE policies ADD COLUMN IF NOT EXISTS max_activations INTEGER;
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS max_activations INTEGER;

CREATE TABLE IF NOT EXISTS license_machines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id UUID NOT NULL REFERENCES licenses (id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    name TEXT,
    hostname TEXT,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_ip TEXT,
    validation_count BIGINT NOT NULL DEFAULT 0,
    deactivated_at TIMESTAMPTZ,
    deactivated_by UUID REFERENCES users (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_license_machines_license_fingerprint
    ON license_machines (license_id, fingerprint);
CREATE INDEX IF NOT EXISTS idx_license_machines_active
    ON license_machines (license_id) WHERE deactivated_at IS NULL;

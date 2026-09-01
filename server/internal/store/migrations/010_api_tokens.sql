CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    role TEXT NOT NULL,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_name ON api_tokens (lower(name));
CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens (token_hash);

ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_role_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_role_check
    CHECK (role IN ('admin', 'operator', 'viewer'));

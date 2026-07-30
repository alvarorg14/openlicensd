CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT,
    duration_days INTEGER,
    expiration_basis TEXT NOT NULL DEFAULT 'on_creation',
    grace_period_days INTEGER NOT NULL DEFAULT 0,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, name)
);

ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_expiration_basis_check;
ALTER TABLE policies ADD CONSTRAINT policies_expiration_basis_check
    CHECK (expiration_basis IN ('on_creation', 'on_first_validation'));

ALTER TABLE licenses ADD COLUMN IF NOT EXISTS product_id UUID;
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS policy_id UUID;
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ;

ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_policy_product_fk;
ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_product_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'licenses'
          AND column_name = 'product_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE licenses ALTER COLUMN product_id SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'licenses'
          AND column_name = 'policy_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE licenses ALTER COLUMN policy_id SET NOT NULL;
    END IF;
END $$;

ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_product_id_fkey;
ALTER TABLE licenses ADD CONSTRAINT licenses_product_id_fkey
    FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE RESTRICT;

ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_id_product_key;
ALTER TABLE policies ADD CONSTRAINT policies_id_product_key UNIQUE (id, product_id);

ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_policy_product_fk;
ALTER TABLE licenses ADD CONSTRAINT licenses_policy_product_fk
    FOREIGN KEY (policy_id, product_id) REFERENCES policies (id, product_id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_licenses_product_id ON licenses (product_id);
CREATE INDEX IF NOT EXISTS idx_licenses_policy_id ON licenses (policy_id);
CREATE INDEX IF NOT EXISTS idx_policies_product_id ON policies (product_id);

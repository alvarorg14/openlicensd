CREATE INDEX IF NOT EXISTS idx_licenses_created_at_id ON licenses (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_licenses_product_created ON licenses (product_id, created_at DESC);

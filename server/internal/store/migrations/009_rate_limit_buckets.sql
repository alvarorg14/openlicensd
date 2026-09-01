CREATE TABLE IF NOT EXISTS rate_limit_buckets (
    scope TEXT NOT NULL,
    bucket_key TEXT NOT NULL,
    tokens DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope, bucket_key)
) WITH (
    fillfactor = 70,
    autovacuum_vacuum_scale_factor = 0.02
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_rate_limit_buckets_updated_at ON rate_limit_buckets (updated_at);

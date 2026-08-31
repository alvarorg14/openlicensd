package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const takeRateLimitTokenSQL = `
WITH locked AS (
    SELECT tokens, updated_at FROM rate_limit_buckets
    WHERE scope = $1 AND bucket_key = $2
    FOR UPDATE
),
refilled AS (
    SELECT LEAST($3::float8, tokens + EXTRACT(EPOCH FROM (NOW() - updated_at)) * $4::float8) AS tokens
    FROM locked
),
updated AS (
    UPDATE rate_limit_buckets b
    SET tokens = CASE WHEN r.tokens >= 1 THEN r.tokens - 1 ELSE r.tokens END,
        updated_at = NOW()
    FROM refilled r
    WHERE b.scope = $1 AND b.bucket_key = $2
    RETURNING r.tokens AS available
)
SELECT available FROM updated
`

// TakeRateLimitToken atomically refills and consumes one token when available.
// The returned available value is the pre-consumption token count; callers treat
// available >= 1 as allowed. When no bucket exists yet, pgx.ErrNoRows is returned.
func (s *Store) TakeRateLimitToken(ctx context.Context, scope, bucketKey string, burst, refillPerSecond float64) (available float64, err error) {
	available, found, err := s.takeRateLimitTokenOnce(ctx, scope, bucketKey, burst, refillPerSecond)
	if err != nil {
		return 0, err
	}
	if found {
		return available, nil
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO rate_limit_buckets (scope, bucket_key, tokens, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (scope, bucket_key) DO NOTHING
	`, scope, bucketKey, burst)
	if err != nil {
		return 0, fmt.Errorf("seed rate limit bucket: %w", err)
	}

	available, found, err = s.takeRateLimitTokenOnce(ctx, scope, bucketKey, burst, refillPerSecond)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, errors.New("rate limit bucket missing after seed")
	}
	return available, nil
}

func (s *Store) takeRateLimitTokenOnce(ctx context.Context, scope, bucketKey string, burst, refillPerSecond float64) (available float64, found bool, err error) {
	err = s.pool.QueryRow(ctx, takeRateLimitTokenSQL, scope, bucketKey, burst, refillPerSecond).Scan(&available)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("take rate limit token: %w", err)
	}
	return available, true, nil
}

// DeleteIdleRateLimitBuckets removes buckets that have not been updated recently.
func (s *Store) DeleteIdleRateLimitBuckets(ctx context.Context, idle time.Duration) (int64, error) {
	cutoff := time.Now().Add(-idle)
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM rate_limit_buckets
		WHERE updated_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete idle rate limit buckets: %w", err)
	}
	return tag.RowsAffected(), nil
}

package ratelimit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/ratelimit"
)

type stubBucketStore struct {
	takeFn   func(ctx context.Context, scope, key string, burst, refillPerSecond float64) (float64, error)
	deleteFn func(ctx context.Context, idle time.Duration) (int64, error)
}

func (s stubBucketStore) TakeRateLimitToken(ctx context.Context, scope, key string, burst, refillPerSecond float64) (float64, error) {
	if s.takeFn != nil {
		return s.takeFn(ctx, scope, key, burst, refillPerSecond)
	}
	return 0, nil
}

func (s stubBucketStore) DeleteIdleRateLimitBuckets(ctx context.Context, idle time.Duration) (int64, error) {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, idle)
	}
	return 0, nil
}

func TestPostgresLimiterFailOpenOnError(t *testing.T) {
	store := stubBucketStore{
		takeFn: func(context.Context, string, string, float64, float64) (float64, error) {
			return 0, errors.New("database unavailable")
		},
	}
	recorder := &stubErrorRecorder{}

	limiter := ratelimit.New(config.RateLimitConfig{
		Enabled:         true,
		Backend:         "postgres",
		PublicPerMinute: 60,
		PublicBurst:     2,
		LoginPerMinute:  30,
		LoginBurst:      1,
		IdleMinutes:     1,
	}, ratelimit.Deps{
		Buckets: store,
		Metrics: recorder,
	})

	allowed, delay := limiter.Allow(context.Background(), ratelimit.ScopePublic, "1.2.3.4")
	if !allowed || delay != 0 {
		t.Fatalf("expected fail-open, allowed=%v delay=%s", allowed, delay)
	}
	if recorder.count != 1 || recorder.scope != "public" {
		t.Fatalf("recorder=%+v", recorder)
	}
}

func TestPostgresLimiterDeniesWhenNoTokens(t *testing.T) {
	store := stubBucketStore{
		takeFn: func(context.Context, string, string, float64, float64) (float64, error) {
			return 0, nil
		},
	}

	limiter := ratelimit.New(config.RateLimitConfig{
		Enabled:         true,
		Backend:         "postgres",
		PublicPerMinute: 60,
		PublicBurst:     2,
		LoginPerMinute:  30,
		LoginBurst:      1,
		IdleMinutes:     1,
	}, ratelimit.Deps{Buckets: store})

	allowed, delay := limiter.Allow(context.Background(), ratelimit.ScopePublic, "1.2.3.4")
	if allowed {
		t.Fatal("expected request to be denied")
	}
	if delay != time.Second {
		t.Fatalf("delay=%s want 1s", delay)
	}
}

func TestPostgresLimiterRetryAfterFromAvailable(t *testing.T) {
	store := stubBucketStore{
		takeFn: func(context.Context, string, string, float64, float64) (float64, error) {
			return 0.5, nil
		},
	}

	limiter := ratelimit.New(config.RateLimitConfig{
		Enabled:         true,
		Backend:         "postgres",
		PublicPerMinute: 60,
		PublicBurst:     2,
		LoginPerMinute:  30,
		LoginBurst:      1,
		IdleMinutes:     1,
	}, ratelimit.Deps{Buckets: store})

	allowed, delay := limiter.Allow(context.Background(), ratelimit.ScopePublic, "1.2.3.4")
	if allowed {
		t.Fatal("expected request to be denied")
	}
	if delay != 500*time.Millisecond {
		t.Fatalf("delay=%s want 500ms", delay)
	}
}

type stubErrorRecorder struct {
	count  int
	scope  string
}

func (s *stubErrorRecorder) RecordRateLimitError(scope string) {
	s.count++
	s.scope = scope
}

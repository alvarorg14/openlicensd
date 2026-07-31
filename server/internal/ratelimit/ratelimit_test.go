package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/ratelimit"
)

func testRateLimitConfig() config.RateLimitConfig {
	return config.RateLimitConfig{
		Enabled:          true,
		PublicPerMinute:  60,
		PublicBurst:      2,
		LoginPerMinute:   30,
		LoginBurst:       1,
		IdleMinutes:      1,
	}
}

func TestLimiterDisabledAlwaysAllows(t *testing.T) {
	limiter := ratelimit.New(config.RateLimitConfig{Enabled: false})
	for i := 0; i < 5; i++ {
		allowed, delay := limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
		if !allowed || delay != 0 {
			t.Fatalf("request %d allowed=%v delay=%s", i, allowed, delay)
		}
	}
}

func TestLimiterBurstThenDeny(t *testing.T) {
	limiter := ratelimit.New(testRateLimitConfig())

	for i := 0; i < 2; i++ {
		allowed, delay := limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
		if !allowed || delay != 0 {
			t.Fatalf("burst request %d allowed=%v delay=%s", i, allowed, delay)
		}
	}

	allowed, delay := limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
	if allowed {
		t.Fatal("expected request to be denied after burst")
	}
	if delay <= 0 {
		t.Fatalf("expected positive retry delay, got %s", delay)
	}
}

func TestLimiterScopesAreIsolated(t *testing.T) {
	limiter := ratelimit.New(testRateLimitConfig())

	allowed, delay := limiter.Allow(ratelimit.ScopeLogin, "1.2.3.4")
	if !allowed || delay != 0 {
		t.Fatalf("login scope allowed=%v delay=%s", allowed, delay)
	}

	allowed, _ = limiter.Allow(ratelimit.ScopeLogin, "1.2.3.4")
	if allowed {
		t.Fatal("expected login scope to deny second request")
	}

	allowed, delay = limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
	if !allowed || delay != 0 {
		t.Fatalf("public scope should remain available, allowed=%v delay=%s", allowed, delay)
	}
}

func TestLimiterRefillsAfterWait(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.PublicPerMinute = 120
	cfg.PublicBurst = 1
	limiter := ratelimit.New(cfg)

	allowed, _ := limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}

	allowed, _ = limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
	if allowed {
		t.Fatal("expected second request to be denied")
	}

	time.Sleep(600 * time.Millisecond)

	allowed, delay := limiter.Allow(ratelimit.ScopePublic, "1.2.3.4")
	if !allowed || delay != 0 {
		t.Fatalf("expected refill after wait, allowed=%v delay=%s", allowed, delay)
	}
}

func TestLimiterRunExitsOnCancel(t *testing.T) {
	limiter := ratelimit.New(testRateLimitConfig())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		limiter.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	if got := ratelimit.RetryAfterSeconds(0); got != 1 {
		t.Fatalf("RetryAfterSeconds(0)=%d want 1", got)
	}
	if got := ratelimit.RetryAfterSeconds(1500 * time.Millisecond); got != 2 {
		t.Fatalf("RetryAfterSeconds(1500ms)=%d want 2", got)
	}
}

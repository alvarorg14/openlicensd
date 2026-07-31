package ratelimit

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openlicensd/openlicensd/server/internal/config"
	"golang.org/x/time/rate"
)

// Scope identifies a rate limit bucket family.
type Scope string

const (
	ScopePublic Scope = "public"
	ScopeLogin  Scope = "login"
)

type scopeConfig struct {
	limit rate.Limit
	burst int
}

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter enforces per-scope, per-key token bucket limits.
type Limiter struct {
	enabled bool
	scopes  map[Scope]scopeConfig
	idle    time.Duration
	mu      sync.Mutex
	buckets map[string]*bucket
}

// New builds a limiter from configuration.
func New(cfg config.RateLimitConfig) *Limiter {
	if !cfg.Enabled {
		return &Limiter{enabled: false}
	}

	return &Limiter{
		enabled: true,
		scopes: map[Scope]scopeConfig{
			ScopePublic: {
				limit: rate.Limit(float64(cfg.PublicPerMinute) / 60.0),
				burst: cfg.PublicBurst,
			},
			ScopeLogin: {
				limit: rate.Limit(float64(cfg.LoginPerMinute) / 60.0),
				burst: cfg.LoginBurst,
			},
		},
		idle:    time.Duration(cfg.IdleMinutes) * time.Minute,
		buckets: make(map[string]*bucket),
	}
}

// Allow reports whether a request is allowed for the given scope and key.
// When denied, the second return value is the suggested retry delay.
func (l *Limiter) Allow(scope Scope, key string) (bool, time.Duration) {
	if l == nil || !l.enabled {
		return true, 0
	}

	scopeCfg, ok := l.scopes[scope]
	if !ok {
		return true, 0
	}

	bucketKey := string(scope) + ":" + key
	now := time.Now()

	l.mu.Lock()
	b, ok := l.buckets[bucketKey]
	if !ok {
		b = &bucket{
			limiter: rate.NewLimiter(scopeCfg.limit, scopeCfg.burst),
		}
		l.buckets[bucketKey] = b
	}
	b.lastSeen = now
	limiter := b.limiter
	l.mu.Unlock()

	if limiter.Allow() {
		return true, 0
	}

	reservation := limiter.Reserve()
	delay := reservation.DelayFrom(now)
	reservation.CancelAt(now)
	return false, delay
}

// Run periodically evicts idle buckets until the context is canceled.
func (l *Limiter) Run(ctx context.Context) {
	if l == nil || !l.enabled {
		return
	}

	interval := l.idle
	if interval < time.Minute {
		interval = time.Minute
	}

	l.evict(time.Now())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			l.evict(now)
		}
	}
}

func (l *Limiter) evict(now time.Time) {
	cutoff := now.Add(-l.idle)

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, b := range l.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}

// RetryAfterSeconds converts a retry delay into an HTTP Retry-After value.
func RetryAfterSeconds(delay time.Duration) int {
	seconds := int((delay + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

// LogStartup logs whether rate limiting is enabled.
func LogStartup(cfg config.RateLimitConfig) {
	if !cfg.Enabled {
		log.Printf("rate limiting disabled")
		return
	}
	log.Printf(
		"rate limiting enabled (public=%d/min burst=%d, login=%d/min burst=%d, idle=%dm)",
		cfg.PublicPerMinute,
		cfg.PublicBurst,
		cfg.LoginPerMinute,
		cfg.LoginBurst,
		cfg.IdleMinutes,
	)
}

package ratelimit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/config"
	"golang.org/x/time/rate"
)

// Scope identifies a rate limit bucket family.
type Scope string

const (
	ScopePublic Scope = "public"
	ScopeLogin  Scope = "login"
)

// Limiter enforces per-scope, per-key token bucket limits.
type Limiter interface {
	Allow(ctx context.Context, scope Scope, key string) (bool, time.Duration)
	Run(ctx context.Context)
}

// ErrorRecorder records rate limit backend failures for observability.
type ErrorRecorder interface {
	RecordRateLimitError(scope string)
}

// BucketStore persists shared rate limit buckets for multi-replica deployments.
type BucketStore interface {
	TakeRateLimitToken(ctx context.Context, scope, key string, burst, refillPerSecond float64) (available float64, err error)
	DeleteIdleRateLimitBuckets(ctx context.Context, idle time.Duration) (int64, error)
}

// Deps supplies optional dependencies for distributed backends.
type Deps struct {
	Buckets BucketStore
	Logger  *slog.Logger
	Metrics ErrorRecorder
}

type scopeConfig struct {
	limit rate.Limit
	burst int
}

type memoryScopeConfig struct {
	burst           int
	refillPerSecond float64
}

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type memoryLimiter struct {
	enabled bool
	scopes  map[Scope]scopeConfig
	idle    time.Duration
	mu      sync.Mutex
	buckets map[string]*bucket
}

// New builds a limiter from configuration and optional dependencies.
func New(cfg config.RateLimitConfig, deps Deps) Limiter {
	switch cfg.Backend {
	case "postgres":
		return NewPostgres(cfg, deps)
	default:
		return NewMemory(cfg)
	}
}

// NewMemory builds an in-process limiter from configuration.
func NewMemory(cfg config.RateLimitConfig) Limiter {
	if !cfg.Enabled {
		return &memoryLimiter{enabled: false}
	}

	return &memoryLimiter{
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
func (l *memoryLimiter) Allow(_ context.Context, scope Scope, key string) (bool, time.Duration) {
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
func (l *memoryLimiter) Run(ctx context.Context) {
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

func (l *memoryLimiter) evict(now time.Time) {
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
func LogStartup(logger *slog.Logger, cfg config.RateLimitConfig) {
	if !cfg.Enabled {
		logger.Info("rate limiting disabled")
		return
	}
	logger.Info("rate limiting enabled",
		slog.String("backend", cfg.Backend),
		slog.Int("public_per_minute", cfg.PublicPerMinute),
		slog.Int("public_burst", cfg.PublicBurst),
		slog.Int("login_per_minute", cfg.LoginPerMinute),
		slog.Int("login_burst", cfg.LoginBurst),
		slog.Int("idle_minutes", cfg.IdleMinutes),
	)
}

func perMinuteToRefillPerSecond(perMinute int) float64 {
	return float64(perMinute) / 60.0
}

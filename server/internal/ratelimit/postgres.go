package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
)

type postgresLimiter struct {
	enabled bool
	scopes  map[Scope]memoryScopeConfig
	idle    time.Duration
	buckets BucketStore
	logger  *slog.Logger
	metrics ErrorRecorder
}

// NewPostgres builds a Postgres-backed limiter for multi-replica deployments.
func NewPostgres(cfg config.RateLimitConfig, deps Deps) Limiter {
	if !cfg.Enabled {
		return &postgresLimiter{enabled: false}
	}
	if deps.Buckets == nil {
		if deps.Logger != nil {
			deps.Logger.Warn("postgres rate limit backend requires a bucket store; falling back to memory")
		}
		return NewMemory(cfg)
	}

	return &postgresLimiter{
		enabled: true,
		scopes: map[Scope]memoryScopeConfig{
			ScopePublic: {
				burst:           cfg.PublicBurst,
				refillPerSecond: perMinuteToRefillPerSecond(cfg.PublicPerMinute),
			},
			ScopeLogin: {
				burst:           cfg.LoginBurst,
				refillPerSecond: perMinuteToRefillPerSecond(cfg.LoginPerMinute),
			},
		},
		idle:    time.Duration(cfg.IdleMinutes) * time.Minute,
		buckets: deps.Buckets,
		logger:  deps.Logger,
		metrics: deps.Metrics,
	}
}

func (l *postgresLimiter) Allow(ctx context.Context, scope Scope, key string) (bool, time.Duration) {
	if l == nil || !l.enabled {
		return true, 0
	}

	scopeCfg, ok := l.scopes[scope]
	if !ok {
		return true, 0
	}

	available, err := l.buckets.TakeRateLimitToken(ctx, string(scope), key, float64(scopeCfg.burst), scopeCfg.refillPerSecond)
	if err != nil {
		logger := logging.FromContext(ctx)
		if logger == nil {
			logger = l.logger
		}
		if logger != nil {
			logger.Warn("rate limit check failed, allowing request",
				slog.Any("err", err),
				slog.String("scope", string(scope)),
			)
		}
		if l.metrics != nil {
			l.metrics.RecordRateLimitError(string(scope))
		}
		return true, 0
	}

	if available >= 1 {
		return true, 0
	}

	delay := retryDelay(available, scopeCfg.refillPerSecond)
	return false, delay
}

func retryDelay(available, refillPerSecond float64) time.Duration {
	if refillPerSecond <= 0 {
		return time.Second
	}
	seconds := (1.0 - available) / refillPerSecond
	if seconds <= 0 {
		return time.Second
	}
	return time.Duration(seconds * float64(time.Second))
}

func (l *postgresLimiter) Run(ctx context.Context) {
	if l == nil || !l.enabled || l.buckets == nil {
		return
	}

	interval := l.idle
	if interval < time.Minute {
		interval = time.Minute
	}

	l.prune(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.prune(ctx)
		}
	}
}

func (l *postgresLimiter) prune(ctx context.Context) {
	removed, err := l.buckets.DeleteIdleRateLimitBuckets(ctx, l.idle)
	if err != nil {
		if l.logger != nil {
			l.logger.Error("rate limit bucket prune failed", slog.Any("err", err))
		}
		return
	}
	if removed > 0 && l.logger != nil {
		l.logger.Info("rate limit bucket prune completed", slog.Int64("removed", removed))
	}
}

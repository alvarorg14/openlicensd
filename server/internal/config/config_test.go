package config_test

import (
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/config"
)

func TestLoadHarborDisabledByDefault(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_HARBOR_ENABLED", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Harbor.Enabled {
		t.Fatalf("expected harbor to be disabled by default")
	}
}

func TestLoadHarborEnabledRequiresConfig(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_HARBOR_ENABLED", "true")
	t.Setenv("OPENLICENSD_HARBOR_URL", "")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_USERNAME", "")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_PASSWORD", "")
	t.Setenv("OPENLICENSD_HARBOR_PROJECTS", "")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error when harbor enabled without required config")
	}
}

func TestLoadHarborEnabledParsesProjects(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_HARBOR_ENABLED", "true")
	t.Setenv("OPENLICENSD_HARBOR_URL", "https://harbor.example.com")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_USERNAME", "admin")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_PASSWORD", "secret")
	t.Setenv("OPENLICENSD_HARBOR_PROJECTS", "project-a, project-b")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Harbor.Enabled {
		t.Fatalf("expected harbor enabled")
	}
	if len(cfg.Harbor.Projects) != 2 || cfg.Harbor.Projects[0] != "project-a" || cfg.Harbor.Projects[1] != "project-b" {
		t.Fatalf("projects=%v", cfg.Harbor.Projects)
	}
	if cfg.Harbor.RobotDurationDays != 1 {
		t.Fatalf("duration=%d want 1", cfg.Harbor.RobotDurationDays)
	}
	if cfg.Harbor.RobotNamePrefix != "openlicensd" {
		t.Fatalf("prefix=%q", cfg.Harbor.RobotNamePrefix)
	}
}

func TestLoadHarborEnabledInvalidDuration(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_HARBOR_ENABLED", "true")
	t.Setenv("OPENLICENSD_HARBOR_URL", "https://harbor.example.com")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_USERNAME", "admin")
	t.Setenv("OPENLICENSD_HARBOR_ADMIN_PASSWORD", "secret")
	t.Setenv("OPENLICENSD_HARBOR_PROJECTS", "project-a")
	t.Setenv("OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid duration")
	}
}

func TestLoadHarborBoolParsing(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
		t.Setenv("OPENLICENSD_HARBOR_ENABLED", "true")
		t.Setenv("OPENLICENSD_HARBOR_URL", "https://harbor.example.com")
		t.Setenv("OPENLICENSD_HARBOR_ADMIN_USERNAME", "admin")
		t.Setenv("OPENLICENSD_HARBOR_ADMIN_PASSWORD", "secret")
		t.Setenv("OPENLICENSD_HARBOR_PROJECTS", "myproject")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if !cfg.Harbor.Enabled {
			t.Fatalf("enabled=false want true")
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
		t.Setenv("OPENLICENSD_HARBOR_ENABLED", "false")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.Harbor.Enabled {
			t.Fatalf("enabled=true want false")
		}
	})
}

func TestLoadSessionDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_COOKIE_SECURE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.SessionTTLHours != 24 {
		t.Fatalf("session ttl=%d want 24", cfg.SessionTTLHours)
	}
	if cfg.SessionCleanupIntervalMinutes != 60 {
		t.Fatalf("session cleanup interval=%d want 60", cfg.SessionCleanupIntervalMinutes)
	}
	if cfg.SessionCleanupInterval() != 60*time.Minute {
		t.Fatalf("session cleanup interval duration=%s want 1h0m0s", cfg.SessionCleanupInterval())
	}
	if !cfg.CookieSecure {
		t.Fatalf("cookie secure=false want true")
	}
	if cfg.BootstrapAdmin.Name != "Administrator" {
		t.Fatalf("bootstrap name=%q", cfg.BootstrapAdmin.Name)
	}
}

func TestLoadSessionCleanupDisabled(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES", "0")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.SessionCleanupIntervalMinutes != 0 {
		t.Fatalf("session cleanup interval=%d want 0", cfg.SessionCleanupIntervalMinutes)
	}
	if cfg.SessionCleanupInterval() != 0 {
		t.Fatalf("session cleanup interval duration=%s want 0", cfg.SessionCleanupInterval())
	}
}

func TestLoadSessionCleanupInvalidInterval(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES", "-1")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for negative session cleanup interval")
	}
}

func TestLoadOIDCDisabledByDefault(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_OIDC_ENABLED", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.OIDC.Enabled {
		t.Fatalf("expected oidc disabled by default")
	}
	if !cfg.LocalLoginEnabled {
		t.Fatalf("expected local login enabled by default")
	}
}

func TestLoadOIDCEnabledRequiresConfig(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_OIDC_ENABLED", "true")
	t.Setenv("OPENLICENSD_OIDC_ISSUER_URL", "")
	t.Setenv("OPENLICENSD_OIDC_CLIENT_ID", "")
	t.Setenv("OPENLICENSD_OIDC_CLIENT_SECRET", "")
	t.Setenv("OPENLICENSD_OIDC_REDIRECT_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error when oidc enabled without required config")
	}
}

func TestLoadOIDCDefaultScopes(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_OIDC_ENABLED", "true")
	t.Setenv("OPENLICENSD_OIDC_ISSUER_URL", "https://issuer.example.com")
	t.Setenv("OPENLICENSD_OIDC_CLIENT_ID", "client-id")
	t.Setenv("OPENLICENSD_OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("OPENLICENSD_OIDC_REDIRECT_URL", "https://app.example.com/api/v1/auth/oidc/callback")
	t.Setenv("OPENLICENSD_OIDC_SCOPES", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.OIDC.Scopes) != 3 || cfg.OIDC.Scopes[0] != "openid" {
		t.Fatalf("scopes=%v", cfg.OIDC.Scopes)
	}
}

func TestLoadLocalLoginDisabledWithoutOIDCFails(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_LOCAL_LOGIN_ENABLED", "false")
	t.Setenv("OPENLICENSD_OIDC_ENABLED", "false")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error when all login methods disabled")
	}
}

func TestOIDCIsAdminEmail(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_OIDC_ADMIN_EMAILS", "Admin@Example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.OIDC.IsAdminEmail("admin@example.com") {
		t.Fatalf("expected admin email match")
	}
	if cfg.OIDC.IsAdminEmail("other@example.com") {
		t.Fatalf("unexpected admin email match")
	}
}

func TestLoadRateLimitDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_RATE_LIMIT_ENABLED", "")
	t.Setenv("OPENLICENSD_TRUSTED_PROXIES", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.RateLimit.Enabled {
		t.Fatalf("expected rate limiting enabled by default")
	}
	if cfg.RateLimit.PublicPerMinute != 600 {
		t.Fatalf("public per minute=%d want 600", cfg.RateLimit.PublicPerMinute)
	}
	if cfg.RateLimit.PublicBurst != 60 {
		t.Fatalf("public burst=%d want 60", cfg.RateLimit.PublicBurst)
	}
	if cfg.RateLimit.LoginPerMinute != 30 {
		t.Fatalf("login per minute=%d want 30", cfg.RateLimit.LoginPerMinute)
	}
	if cfg.RateLimit.LoginBurst != 10 {
		t.Fatalf("login burst=%d want 10", cfg.RateLimit.LoginBurst)
	}
	if cfg.RateLimit.IdleMinutes != 10 {
		t.Fatalf("idle minutes=%d want 10", cfg.RateLimit.IdleMinutes)
	}
	if cfg.RateLimit.Backend != "memory" {
		t.Fatalf("backend=%q want memory", cfg.RateLimit.Backend)
	}
}

func TestLoadRateLimitBackendPostgres(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_RATE_LIMIT_BACKEND", "postgres")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RateLimit.Backend != "postgres" {
		t.Fatalf("backend=%q want postgres", cfg.RateLimit.Backend)
	}
}

func TestLoadRateLimitInvalidBackend(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_RATE_LIMIT_BACKEND", "redis")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid rate limit backend")
	}
}

func TestLoadRateLimitInvalidPublicBurst(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_RATE_LIMIT_PUBLIC_BURST", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid public burst")
	}
}

func TestLoadTrustedProxiesInvalidEntry(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_TRUSTED_PROXIES", "not-an-ip")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid trusted proxy")
	}
}

func TestLoadTrustedProxiesParsesCSV(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_TRUSTED_PROXIES", "10.0.0.0/8, 10.1.2.3")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Fatalf("trusted proxies=%v", cfg.TrustedProxies)
	}
	if cfg.TrustedProxies[0] != "10.0.0.0/8" || cfg.TrustedProxies[1] != "10.1.2.3" {
		t.Fatalf("trusted proxies=%v", cfg.TrustedProxies)
	}
}

func TestLoadLogDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_LOG_LEVEL", "")
	t.Setenv("OPENLICENSD_LOG_FORMAT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("log level=%q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("log format=%q", cfg.Log.Format)
	}
}

func TestLoadLogInvalidLevel(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_LOG_LEVEL", "trace")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid log level")
	}
}

func TestLoadLogInvalidFormat(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_LOG_FORMAT", "xml")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for invalid log format")
	}
}

func TestLoadMetricsDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_METRICS_ENABLED", "")
	t.Setenv("OPENLICENSD_METRICS_ADDR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Metrics.Enabled {
		t.Fatalf("expected metrics enabled by default")
	}
	if cfg.Metrics.Addr != ":9090" {
		t.Fatalf("metrics addr=%q want :9090", cfg.Metrics.Addr)
	}
}

func TestLoadMetricsDisabled(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_METRICS_ENABLED", "false")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Metrics.Enabled {
		t.Fatalf("expected metrics disabled")
	}
}

func TestLoadMetricsEmptyAddr(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_METRICS_ENABLED", "true")
	t.Setenv("OPENLICENSD_METRICS_ADDR", "   ")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for empty metrics addr")
	}
}

func TestLoadMetricsAddrEqualsAPIAddr(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_ADDR", ":8080")
	t.Setenv("OPENLICENSD_METRICS_ENABLED", "true")
	t.Setenv("OPENLICENSD_METRICS_ADDR", ":8080")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error when metrics addr equals API addr")
	}
}

func TestLoadDatabaseDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONNS", "")
	t.Setenv("OPENLICENSD_DATABASE_MIN_CONNS", "")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES", "")
	t.Setenv("OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Database.MaxConns != 0 {
		t.Fatalf("max conns=%d want 0", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 0 {
		t.Fatalf("min conns=%d want 0", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnIdleMinutes != 0 {
		t.Fatalf("idle minutes=%d want 0", cfg.Database.MaxConnIdleMinutes)
	}
	if cfg.Database.StatementTimeoutSeconds != 0 {
		t.Fatalf("statement timeout=%d want 0", cfg.Database.StatementTimeoutSeconds)
	}
}

func TestLoadDatabaseOverrides(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONNS", "20")
	t.Setenv("OPENLICENSD_DATABASE_MIN_CONNS", "2")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES", "15")
	t.Setenv("OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS", "60")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Database.MaxConns != 20 {
		t.Fatalf("max conns=%d want 20", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 2 {
		t.Fatalf("min conns=%d want 2", cfg.Database.MinConns)
	}
	if cfg.Database.MaxConnIdleMinutes != 15 {
		t.Fatalf("idle minutes=%d want 15", cfg.Database.MaxConnIdleMinutes)
	}
	if cfg.Database.StatementTimeoutSeconds != 60 {
		t.Fatalf("statement timeout=%d want 60", cfg.Database.StatementTimeoutSeconds)
	}
}

func TestLoadDatabaseNegativeMaxConns(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONNS", "-1")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for negative max conns")
	}
}

func TestLoadDatabaseMinExceedsMax(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_DATABASE_MAX_CONNS", "5")
	t.Setenv("OPENLICENSD_DATABASE_MIN_CONNS", "10")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error when min conns exceeds max conns")
	}
}

func TestLoadRequestTimeoutDefaults(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_REQUEST_TIMEOUT_SECONDS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RequestTimeoutSeconds != 30 {
		t.Fatalf("request timeout=%d want 30", cfg.RequestTimeoutSeconds)
	}
	if cfg.RequestTimeout() != 30*time.Second {
		t.Fatalf("request timeout duration=%s want 30s", cfg.RequestTimeout())
	}
}

func TestLoadRequestTimeoutOverride(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_REQUEST_TIMEOUT_SECONDS", "45")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RequestTimeoutSeconds != 45 {
		t.Fatalf("request timeout=%d want 45", cfg.RequestTimeoutSeconds)
	}
}

func TestLoadRequestTimeoutDisabled(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_REQUEST_TIMEOUT_SECONDS", "0")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.RequestTimeoutSeconds != 0 {
		t.Fatalf("request timeout=%d want 0", cfg.RequestTimeoutSeconds)
	}
	if cfg.RequestTimeout() != 0 {
		t.Fatalf("request timeout duration=%s want 0", cfg.RequestTimeout())
	}
}

func TestLoadRequestTimeoutNegative(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_URL", "postgres://example")
	t.Setenv("OPENLICENSD_REQUEST_TIMEOUT_SECONDS", "-1")

	_, err := config.Load()
	if err == nil {
		t.Fatalf("expected error for negative request timeout")
	}
}

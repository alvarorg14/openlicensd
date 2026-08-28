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

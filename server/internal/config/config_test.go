package config_test

import (
	"testing"

	"github.com/openlicensd/openlicensd/server/internal/config"
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

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.SessionTTLHours != 24 {
		t.Fatalf("session ttl=%d want 24", cfg.SessionTTLHours)
	}
	if !cfg.CookieSecure {
		t.Fatalf("cookie secure=false want true")
	}
	if cfg.BootstrapAdmin.Name != "Administrator" {
		t.Fatalf("bootstrap name=%q", cfg.BootstrapAdmin.Name)
	}
}

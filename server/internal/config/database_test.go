package config_test

import (
	"strings"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/config"
)

func setRequiredDatabaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OPENLICENSD_DATABASE_HOST", "localhost")
	t.Setenv("OPENLICENSD_DATABASE_USER", "openlicensd")
	t.Setenv("OPENLICENSD_DATABASE_NAME", "openlicensd")
}

func TestDatabaseConnectionStringDefaults(t *testing.T) {
	conn := config.DatabaseConfig{
		Host:    "db.example.com",
		Port:    5432,
		User:    "app",
		Name:    "licenses",
		SSLMode: "require",
	}

	got := conn.ConnectionString()
	wantParts := []string{
		"host=db.example.com",
		"port=5432",
		"user=app",
		"dbname=licenses",
		"sslmode=require",
	}
	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Fatalf("connection string %q missing %q", got, part)
		}
	}
	if strings.Contains(got, "password=") {
		t.Fatalf("connection string %q should omit empty password", got)
	}
}

func TestDatabaseConnectionStringPasswordAndOptions(t *testing.T) {
	conn := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5433,
		User:     "app",
		Password: "pa'ss",
		Name:     "openlicensd",
		SSLMode:  "disable",
		Options:  "connect_timeout=5 application_name=openlicensd",
	}

	got := conn.ConnectionString()
	if !strings.Contains(got, "password='pa''ss'") {
		t.Fatalf("connection string %q should quote password", got)
	}
	if !strings.Contains(got, "connect_timeout=5") || !strings.Contains(got, "application_name=openlicensd") {
		t.Fatalf("connection string %q should include options", got)
	}
}

func TestLoadRequiresDatabaseHostUserName(t *testing.T) {
	t.Setenv("OPENLICENSD_DATABASE_HOST", "")
	t.Setenv("OPENLICENSD_DATABASE_USER", "")
	t.Setenv("OPENLICENSD_DATABASE_NAME", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when database connection fields are missing")
	}
	if !strings.Contains(err.Error(), "OPENLICENSD_DATABASE_HOST is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadBuildsDatabaseURLFromDiscreteVars(t *testing.T) {
	setRequiredDatabaseEnv(t)
	t.Setenv("OPENLICENSD_DATABASE_PORT", "5433")
	t.Setenv("OPENLICENSD_DATABASE_PASSWORD", "secret")
	t.Setenv("OPENLICENSD_DATABASE_SSLMODE", "disable")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !strings.Contains(cfg.DatabaseURL, "host=localhost") {
		t.Fatalf("database url=%q", cfg.DatabaseURL)
	}
	if !strings.Contains(cfg.DatabaseURL, "port=5433") {
		t.Fatalf("database url=%q", cfg.DatabaseURL)
	}
	if !strings.Contains(cfg.DatabaseURL, "password=secret") {
		t.Fatalf("database url=%q", cfg.DatabaseURL)
	}
	if !strings.Contains(cfg.DatabaseURL, "sslmode=disable") {
		t.Fatalf("database url=%q", cfg.DatabaseURL)
	}
}

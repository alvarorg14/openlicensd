package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type HarborConfig struct {
	Enabled            bool
	URL                string
	AdminUsername      string
	AdminPassword      string
	Projects           []string
	RobotDurationDays  int
	RobotNamePrefix    string
	InsecureSkipVerify bool
	Debug              bool
}

type Config struct {
	Addr              string
	DatabaseURL       string
	AdminUser         string
	AdminPasswordHash string
	JWTSecret         string
	Harbor            HarborConfig
}

func Load() (*Config, error) {
	cfg := &Config{
		Addr:              getEnv("OPENLICENSD_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("OPENLICENSD_DATABASE_URL"),
		AdminUser:         os.Getenv("OPENLICENSD_ADMIN_USER"),
		AdminPasswordHash: os.Getenv("OPENLICENSD_ADMIN_PASSWORD_HASH"),
		JWTSecret:         os.Getenv("OPENLICENSD_JWT_SECRET"),
		Harbor: HarborConfig{
			Enabled:            getBoolEnv("OPENLICENSD_HARBOR_ENABLED", false),
			URL:                os.Getenv("OPENLICENSD_HARBOR_URL"),
			AdminUsername:      os.Getenv("OPENLICENSD_HARBOR_ADMIN_USERNAME"),
			AdminPassword:      os.Getenv("OPENLICENSD_HARBOR_ADMIN_PASSWORD"),
			Projects:           parseCSV(os.Getenv("OPENLICENSD_HARBOR_PROJECTS")),
			RobotDurationDays:  getIntEnv("OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS", 1),
			RobotNamePrefix:    getEnv("OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX", "openlicensd"),
			InsecureSkipVerify: getBoolEnv("OPENLICENSD_HARBOR_INSECURE_SKIP_VERIFY", false),
			Debug:              getBoolEnv("OPENLICENSD_HARBOR_DEBUG", false),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("OPENLICENSD_DATABASE_URL is required")
	}
	if cfg.AdminUser == "" {
		return nil, fmt.Errorf("OPENLICENSD_ADMIN_USER is required")
	}
	if cfg.AdminPasswordHash == "" {
		return nil, fmt.Errorf("OPENLICENSD_ADMIN_PASSWORD_HASH is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("OPENLICENSD_JWT_SECRET is required")
	}

	if err := cfg.Harbor.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (h HarborConfig) validate() error {
	if !h.Enabled {
		return nil
	}

	if h.URL == "" {
		return fmt.Errorf("OPENLICENSD_HARBOR_URL is required when OPENLICENSD_HARBOR_ENABLED is true")
	}
	if h.AdminUsername == "" {
		return fmt.Errorf("OPENLICENSD_HARBOR_ADMIN_USERNAME is required when OPENLICENSD_HARBOR_ENABLED is true")
	}
	if h.AdminPassword == "" {
		return fmt.Errorf("OPENLICENSD_HARBOR_ADMIN_PASSWORD is required when OPENLICENSD_HARBOR_ENABLED is true")
	}
	if len(h.Projects) == 0 {
		return fmt.Errorf("OPENLICENSD_HARBOR_PROJECTS is required when OPENLICENSD_HARBOR_ENABLED is true")
	}
	if h.RobotDurationDays < 1 {
		return fmt.Errorf("OPENLICENSD_HARBOR_ROBOT_DURATION_DAYS must be at least 1")
	}
	if h.RobotNamePrefix == "" {
		return fmt.Errorf("OPENLICENSD_HARBOR_ROBOT_NAME_PREFIX must not be empty")
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

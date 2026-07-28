package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr             string
	DatabaseURL      string
	AdminUser        string
	AdminPasswordHash string
	JWTSecret        string
}

func Load() (*Config, error) {
	cfg := &Config{
		Addr:              getEnv("OPENLICENSD_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("OPENLICENSD_DATABASE_URL"),
		AdminUser:         os.Getenv("OPENLICENSD_ADMIN_USER"),
		AdminPasswordHash: os.Getenv("OPENLICENSD_ADMIN_PASSWORD_HASH"),
		JWTSecret:         os.Getenv("OPENLICENSD_JWT_SECRET"),
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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

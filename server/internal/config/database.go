package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:                    os.Getenv("OPENLICENSD_DATABASE_HOST"),
		Port:                    getIntEnv("OPENLICENSD_DATABASE_PORT", 5432),
		User:                    os.Getenv("OPENLICENSD_DATABASE_USER"),
		Password:                os.Getenv("OPENLICENSD_DATABASE_PASSWORD"),
		Name:                    os.Getenv("OPENLICENSD_DATABASE_NAME"),
		SSLMode:                 getEnv("OPENLICENSD_DATABASE_SSLMODE", "require"),
		Options:                 os.Getenv("OPENLICENSD_DATABASE_OPTIONS"),
		MaxConns:                getIntEnv("OPENLICENSD_DATABASE_MAX_CONNS", 0),
		MinConns:                getIntEnv("OPENLICENSD_DATABASE_MIN_CONNS", 0),
		MaxConnIdleMinutes:      getIntEnv("OPENLICENSD_DATABASE_MAX_CONN_IDLE_MINUTES", 0),
		StatementTimeoutSeconds: getIntEnv("OPENLICENSD_DATABASE_STATEMENT_TIMEOUT_SECONDS", 0),
	}
}

func (d DatabaseConfig) ConnectionString() string {
	parts := []string{
		"host=" + quoteConnValue(d.Host),
		"port=" + strconv.Itoa(d.Port),
		"user=" + quoteConnValue(d.User),
		"dbname=" + quoteConnValue(d.Name),
		"sslmode=" + quoteConnValue(d.SSLMode),
	}
	if d.Password != "" {
		parts = append(parts, "password="+quoteConnValue(d.Password))
	}
	conn := strings.Join(parts, " ")
	options := strings.TrimSpace(d.Options)
	if options != "" {
		return conn + " " + options
	}
	return conn
}

func quoteConnValue(value string) string {
	if value == "" {
		return "''"
	}
	for _, r := range value {
		if r == ' ' || r == '\'' || r == '\\' {
			return "'" + strings.ReplaceAll(value, `'`, `''`) + "'"
		}
	}
	return value
}

func (d DatabaseConfig) validateConnection() error {
	if strings.TrimSpace(d.Host) == "" {
		return fmt.Errorf("OPENLICENSD_DATABASE_HOST is required")
	}
	if strings.TrimSpace(d.User) == "" {
		return fmt.Errorf("OPENLICENSD_DATABASE_USER is required")
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("OPENLICENSD_DATABASE_NAME is required")
	}
	if d.Port < 1 || d.Port > 65535 {
		return fmt.Errorf("OPENLICENSD_DATABASE_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(d.SSLMode) == "" {
		return fmt.Errorf("OPENLICENSD_DATABASE_SSLMODE must not be empty")
	}
	return nil
}

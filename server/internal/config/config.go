package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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

type OIDCConfig struct {
	Enabled         bool
	IssuerURL       string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	Scopes          []string
	DefaultRole     string
	ProviderName    string
	AdminEmails     []string
}

type BootstrapAdminConfig struct {
	Email        string
	Name         string
	PasswordHash string
}

type Config struct {
	Addr                          string
	DatabaseURL                   string
	BootstrapAdmin                BootstrapAdminConfig
	SessionTTLHours               int
	SessionCleanupIntervalMinutes int
	CookieSecure                  bool
	LocalLoginEnabled             bool
	Harbor                        HarborConfig
	OIDC                          OIDCConfig
}

func Load() (*Config, error) {
	localLoginEnabled := getBoolEnv("OPENLICENSD_LOCAL_LOGIN_ENABLED", true)

	cfg := &Config{
		Addr:        getEnv("OPENLICENSD_ADDR", ":8080"),
		DatabaseURL: os.Getenv("OPENLICENSD_DATABASE_URL"),
		BootstrapAdmin: BootstrapAdminConfig{
			Email:        os.Getenv("OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL"),
			Name:         getEnv("OPENLICENSD_BOOTSTRAP_ADMIN_NAME", "Administrator"),
			PasswordHash: os.Getenv("OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH"),
		},
		SessionTTLHours:               getIntEnv("OPENLICENSD_SESSION_TTL_HOURS", 24),
		SessionCleanupIntervalMinutes: getIntEnv("OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES", 60),
		CookieSecure:                  getBoolEnv("OPENLICENSD_COOKIE_SECURE", true),
		LocalLoginEnabled: localLoginEnabled,
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
		OIDC: OIDCConfig{
			Enabled:           getBoolEnv("OPENLICENSD_OIDC_ENABLED", false),
			IssuerURL:         os.Getenv("OPENLICENSD_OIDC_ISSUER_URL"),
			ClientID:          os.Getenv("OPENLICENSD_OIDC_CLIENT_ID"),
			ClientSecret:      os.Getenv("OPENLICENSD_OIDC_CLIENT_SECRET"),
			RedirectURL:       os.Getenv("OPENLICENSD_OIDC_REDIRECT_URL"),
			Scopes:            parseOIDCScopes(os.Getenv("OPENLICENSD_OIDC_SCOPES")),
			DefaultRole:       getEnv("OPENLICENSD_OIDC_DEFAULT_ROLE", "viewer"),
			ProviderName:      getEnv("OPENLICENSD_OIDC_PROVIDER_NAME", "SSO"),
			AdminEmails:       parseLowerCSV(os.Getenv("OPENLICENSD_OIDC_ADMIN_EMAILS")),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("OPENLICENSD_DATABASE_URL is required")
	}
	if cfg.SessionTTLHours < 1 {
		return nil, fmt.Errorf("OPENLICENSD_SESSION_TTL_HOURS must be at least 1")
	}
	if cfg.SessionCleanupIntervalMinutes < 0 {
		return nil, fmt.Errorf("OPENLICENSD_SESSION_CLEANUP_INTERVAL_MINUTES must be 0 or greater")
	}

	if err := cfg.Harbor.validate(); err != nil {
		return nil, err
	}
	if err := cfg.OIDC.validate(); err != nil {
		return nil, err
	}
	if !cfg.LocalLoginEnabled && !cfg.OIDC.Enabled {
		return nil, fmt.Errorf("at least one login method must be enabled: set OPENLICENSD_LOCAL_LOGIN_ENABLED or OPENLICENSD_OIDC_ENABLED")
	}

	return cfg, nil
}

func (c *Config) SessionCleanupInterval() time.Duration {
	return time.Duration(c.SessionCleanupIntervalMinutes) * time.Minute
}

func (o OIDCConfig) IsAdminEmail(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, admin := range o.AdminEmails {
		if admin == normalized {
			return true
		}
	}
	return false
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

func (o OIDCConfig) validate() error {
	if !o.Enabled {
		return nil
	}

	if o.IssuerURL == "" {
		return fmt.Errorf("OPENLICENSD_OIDC_ISSUER_URL is required when OPENLICENSD_OIDC_ENABLED is true")
	}
	if o.ClientID == "" {
		return fmt.Errorf("OPENLICENSD_OIDC_CLIENT_ID is required when OPENLICENSD_OIDC_ENABLED is true")
	}
	if o.ClientSecret == "" {
		return fmt.Errorf("OPENLICENSD_OIDC_CLIENT_SECRET is required when OPENLICENSD_OIDC_ENABLED is true")
	}
	if o.RedirectURL == "" {
		return fmt.Errorf("OPENLICENSD_OIDC_REDIRECT_URL is required when OPENLICENSD_OIDC_ENABLED is true")
	}
	if !isValidRole(o.DefaultRole) {
		return fmt.Errorf("OPENLICENSD_OIDC_DEFAULT_ROLE must be admin, operator, or viewer")
	}
	if o.ProviderName == "" {
		return fmt.Errorf("OPENLICENSD_OIDC_PROVIDER_NAME must not be empty")
	}

	return nil
}

func isValidRole(role string) bool {
	switch role {
	case "admin", "operator", "viewer":
		return true
	default:
		return false
	}
}

func parseOIDCScopes(value string) []string {
	scopes := parseCSV(value)
	if len(scopes) == 0 {
		return []string{"openid", "profile", "email"}
	}
	return scopes
}

func parseLowerCSV(value string) []string {
	parts := parseCSV(value)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.ToLower(part))
	}
	return out
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

package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alvarorg14/openlicensd/server/internal/config"
)

func BootstrapAdmin(ctx context.Context, st *Store, cfg *config.Config) error {
	count, err := st.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	if cfg.BootstrapAdmin.Email == "" || cfg.BootstrapAdmin.PasswordHash == "" {
		return fmt.Errorf("no users exist: set OPENLICENSD_BOOTSTRAP_ADMIN_EMAIL and OPENLICENSD_BOOTSTRAP_ADMIN_PASSWORD_HASH to seed the first admin (generate hash with: make hash-password PASSWORD=yourpassword)")
	}

	hash := cfg.BootstrapAdmin.PasswordHash
	_, err = st.CreateUser(ctx, cfg.BootstrapAdmin.Email, cfg.BootstrapAdmin.Name, &hash, RoleAdmin, AuthProviderLocal, nil)
	if err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}

	slog.Default().Info("bootstrap admin seeded", slog.String("email", cfg.BootstrapAdmin.Email))
	return nil
}

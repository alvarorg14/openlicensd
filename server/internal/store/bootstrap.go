package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alvarorg14/openlicensd/server/internal/config"
)

// bootstrapLockID serializes concurrent BootstrapAdmin() callers (multi-replica startup).
const bootstrapLockID int64 = 0x4f50454e424f4f54 // "OPENBOOT"

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

	conn, err := st.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, bootstrapLockID); err != nil {
		return fmt.Errorf("acquire bootstrap lock: %w", err)
	}
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, bootstrapLockID) //nolint:errcheck

	count, err = st.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash := cfg.BootstrapAdmin.PasswordHash
	_, err = st.CreateUser(ctx, cfg.BootstrapAdmin.Email, cfg.BootstrapAdmin.Name, &hash, RoleAdmin, AuthProviderLocal, nil)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			existing, lookupErr := st.GetUserByEmail(ctx, cfg.BootstrapAdmin.Email)
			if lookupErr == nil && existing != nil {
				slog.Default().Info("bootstrap admin already exists", slog.String("email", cfg.BootstrapAdmin.Email))
				return nil
			}
		}
		return fmt.Errorf("create bootstrap admin: %w", err)
	}

	slog.Default().Info("bootstrap admin seeded", slog.String("email", cfg.BootstrapAdmin.Email))
	return nil
}

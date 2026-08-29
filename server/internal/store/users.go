package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

const AuthProviderLocal = "local"
const AuthProviderOIDC = "oidc"

type User struct {
	ID                  uuid.UUID
	Email               string
	Name                string
	PasswordHash        *string
	Role                Role
	AuthProvider        string
	ExternalID          *string
	PictureURL          *string
	DisabledAt          *time.Time
	FailedLoginAttempts int
	LockedUntil         *time.Time
	LastLoginAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

const userColumns = `
	id, email, name, password_hash, role, auth_provider, external_id, picture_url,
	disabled_at, failed_login_attempts, locked_until, last_login_at, created_at, updated_at
`

func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	const q = `SELECT COUNT(*) FROM users`
	var count int64
	if err := s.pool.QueryRow(ctx, q).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) CreateUser(ctx context.Context, email, name string, passwordHash *string, role Role, authProvider string, externalID *string) (*User, error) {
	if authProvider == "" {
		authProvider = AuthProviderLocal
	}

	const q = `
		INSERT INTO users (email, name, password_hash, role, auth_provider, external_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + userColumns

	row := s.pool.QueryRow(ctx, q, strings.ToLower(strings.TrimSpace(email)), name, passwordHash, role, authProvider, externalID)
	u, err := scanUser(row)
	if err != nil {
		return nil, mapInsertError(err)
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context, params ListParams) ([]User, int64, error) {
	qb := newQueryBuilder()
	if params.Search != "" {
		pattern := searchPattern(params.Search)
		qb.add("(name ILIKE $%d OR email ILIKE $%d)", pattern, pattern)
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "created_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "id")

	q := `
		SELECT ` + userColumns + `, COUNT(*) OVER() AS total_count
		FROM users` + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		users      []User
		totalCount int64
	)
	for rows.Next() {
		var u User
		var role string
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &u.AuthProvider, &u.ExternalID, &u.PictureURL,
			&u.DisabledAt, &u.FailedLoginAttempts, &u.LockedUntil, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		u.Role = Role(role)
		users = append(users, u)
	}

	return users, totalCount, rows.Err()
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByExternalID(ctx context.Context, authProvider, externalID string) (*User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE auth_provider = $1 AND external_id = $2
	`

	row := s.pool.QueryRow(ctx, q, authProvider, externalID)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) LinkUserToProvider(ctx context.Context, id uuid.UUID, authProvider, externalID string) (*User, error) {
	const q = `
		UPDATE users
		SET auth_provider = $2, external_id = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userColumns

	row := s.pool.QueryRow(ctx, q, id, authProvider, externalID)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapInsertError(err)
	}
	return u, nil
}

func (s *Store) SyncUserProfile(ctx context.Context, id uuid.UUID, email, name string, pictureURL *string) (*User, error) {
	const q = `
		UPDATE users
		SET email = $2, name = $3, picture_url = COALESCE($4, picture_url), updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userColumns

	row := s.pool.QueryRow(ctx, q, id, strings.ToLower(strings.TrimSpace(email)), name, pictureURL)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapInsertError(err)
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT ` + userColumns + `
		FROM users
		WHERE lower(email) = lower($1)
	`

	row := s.pool.QueryRow(ctx, q, strings.TrimSpace(email))
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, email, name string, role Role) (*User, error) {
	const q = `
		UPDATE users
		SET email = $2, name = $3, role = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userColumns

	row := s.pool.QueryRow(ctx, q, id, strings.ToLower(strings.TrimSpace(email)), name, role)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapInsertError(err)
	}
	return u, nil
}

func (s *Store) SetUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const q = `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`

	tag, err := s.pool.Exec(ctx, q, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) SetUserDisabled(ctx context.Context, id uuid.UUID, disabled bool) (*User, error) {
	var q string
	if disabled {
		q = `
			UPDATE users
			SET disabled_at = NOW(), updated_at = NOW()
			WHERE id = $1
			RETURNING ` + userColumns
	} else {
		q = `
			UPDATE users
			SET disabled_at = NULL, failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
			WHERE id = $1
			RETURNING ` + userColumns
	}

	row := s.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM users WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) RecordFailedLogin(ctx context.Context, id uuid.UUID, maxAttempts int, lockDuration time.Duration) error {
	const q = `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= $2 THEN NOW() + ($3::bigint * interval '1 second')
		        ELSE locked_until
		    END,
		    updated_at = NOW()
		WHERE id = $1
	`

	seconds := int64(lockDuration.Seconds())
	_, err := s.pool.Exec(ctx, q, id, maxAttempts, seconds)
	return err
}

func (s *Store) ClearFailedLogin(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL, last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, q, id)
	return err
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var role string
	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &role, &u.AuthProvider, &u.ExternalID, &u.PictureURL,
		&u.DisabledAt, &u.FailedLoginAttempts, &u.LockedUntil, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Role = Role(role)
	return &u, nil
}

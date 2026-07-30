package store

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type License struct {
	ID               uuid.UUID
	Label            string
	KeyHash          string
	KeyPrefix        string
	ProductID        uuid.UUID
	PolicyID         uuid.UUID
	ExpiresAt        *time.Time
	ActivatedAt      *time.Time
	Revoked          bool
	CreatedAt        time.Time
	LastValidatedAt  *time.Time
	ValidationCount  int64
	ProductCode      string
	ProductName      string
	PolicyName       string
	GracePeriodDays  int
	ExpirationBasis  ExpirationBasis
	DurationDays     *int
}

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

const licenseColumns = `
	l.id, l.label, l.key_hash, l.key_prefix, l.product_id, l.policy_id,
	l.expires_at, l.activated_at, l.revoked, l.created_at, l.last_validated_at, l.validation_count,
	p.code, p.name, pol.name, pol.grace_period_days, pol.expiration_basis, pol.duration_days
`

const licenseFromJoin = `
	FROM licenses l
	JOIN products p ON p.id = l.product_id
	JOIN policies pol ON pol.id = l.policy_id
`

func (s *Store) CreateLicense(ctx context.Context, label, keyHash, keyPrefix string, productID, policyID uuid.UUID, expiresAt *time.Time) (*License, error) {
	const q = `
		INSERT INTO licenses (label, key_hash, key_prefix, product_id, policy_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, q, label, keyHash, keyPrefix, productID, policyID, expiresAt).Scan(&id); err != nil {
		if isReferentialViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}

	return s.GetLicenseByID(ctx, id)
}

func (s *Store) GetLicenseByID(ctx context.Context, id uuid.UUID) (*License, error) {
	const q = `
		SELECT ` + licenseColumns + licenseFromJoin + `
		WHERE l.id = $1
	`

	row := s.pool.QueryRow(ctx, q, id)
	return scanLicense(row)
}

func (s *Store) ListLicenses(ctx context.Context) ([]License, error) {
	const q = `
		SELECT ` + licenseColumns + licenseFromJoin + `
		ORDER BY l.created_at DESC
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLicenses(rows)
}

func (s *Store) GetLicenseByKeyHash(ctx context.Context, keyHash string) (*License, error) {
	const q = `
		SELECT ` + licenseColumns + licenseFromJoin + `
		WHERE l.key_hash = $1
	`

	row := s.pool.QueryRow(ctx, q, keyHash)
	lic, err := scanLicense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return lic, nil
}

func (s *Store) SetLicenseRevoked(ctx context.Context, id uuid.UUID, revoked bool) (*License, error) {
	const q = `
		UPDATE licenses
		SET revoked = $2
		WHERE id = $1
	`

	tag, err := s.pool.Exec(ctx, q, id, revoked)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}

	return s.GetLicenseByID(ctx, id)
}

func (s *Store) UpdateLicense(ctx context.Context, id uuid.UUID, label string, expiresAt *time.Time) (*License, error) {
	const q = `
		UPDATE licenses
		SET label = $2, expires_at = $3
		WHERE id = $1
	`

	tag, err := s.pool.Exec(ctx, q, id, label, expiresAt)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}

	return s.GetLicenseByID(ctx, id)
}

func (s *Store) ActivateLicense(ctx context.Context, id uuid.UUID, expiresAt time.Time) (*License, error) {
	const q = `
		UPDATE licenses
		SET activated_at = NOW(), expires_at = $2
		WHERE id = $1 AND activated_at IS NULL AND expires_at IS NULL
	`

	tag, err := s.pool.Exec(ctx, q, id, expiresAt)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return s.GetLicenseByID(ctx, id)
	}

	return s.GetLicenseByID(ctx, id)
}

func (s *Store) DeleteLicense(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM licenses WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) RecordValidation(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE licenses
		SET last_validated_at = NOW(), validation_count = validation_count + 1
		WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, q, id)
	return err
}

func scanLicense(row pgx.Row) (*License, error) {
	var l License
	var expirationBasis string
	err := row.Scan(
		&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ProductID, &l.PolicyID,
		&l.ExpiresAt, &l.ActivatedAt, &l.Revoked, &l.CreatedAt, &l.LastValidatedAt, &l.ValidationCount,
		&l.ProductCode, &l.ProductName, &l.PolicyName, &l.GracePeriodDays, &expirationBasis, &l.DurationDays,
	)
	if err != nil {
		return nil, err
	}
	l.ExpirationBasis = ExpirationBasis(expirationBasis)
	return &l, nil
}

func scanLicenses(rows pgx.Rows) ([]License, error) {
	var licenses []License
	for rows.Next() {
		var l License
		var expirationBasis string
		if err := rows.Scan(
			&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ProductID, &l.PolicyID,
			&l.ExpiresAt, &l.ActivatedAt, &l.Revoked, &l.CreatedAt, &l.LastValidatedAt, &l.ValidationCount,
			&l.ProductCode, &l.ProductName, &l.PolicyName, &l.GracePeriodDays, &expirationBasis, &l.DurationDays,
		); err != nil {
			return nil, err
		}
		l.ExpirationBasis = ExpirationBasis(expirationBasis)
		licenses = append(licenses, l)
	}
	return licenses, rows.Err()
}

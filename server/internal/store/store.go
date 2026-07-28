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
	ExpiresAt        *time.Time
	Revoked          bool
	CreatedAt        time.Time
	LastValidatedAt  *time.Time
	ValidationCount  int64
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

const licenseColumns = `id, label, key_hash, key_prefix, expires_at, revoked, created_at, last_validated_at, validation_count`

func (s *Store) CreateLicense(ctx context.Context, label, keyHash, keyPrefix string, expiresAt *time.Time) (*License, error) {
	const q = `
		INSERT INTO licenses (label, key_hash, key_prefix, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + licenseColumns

	row := s.pool.QueryRow(ctx, q, label, keyHash, keyPrefix, expiresAt)
	return scanLicense(row)
}

func (s *Store) ListLicenses(ctx context.Context) ([]License, error) {
	const q = `
		SELECT ` + licenseColumns + `
		FROM licenses
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []License
	for rows.Next() {
		var l License
		if err := rows.Scan(
			&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ExpiresAt, &l.Revoked, &l.CreatedAt,
			&l.LastValidatedAt, &l.ValidationCount,
		); err != nil {
			return nil, err
		}
		licenses = append(licenses, l)
	}

	return licenses, rows.Err()
}

func (s *Store) GetLicenseByKeyHash(ctx context.Context, keyHash string) (*License, error) {
	const q = `
		SELECT ` + licenseColumns + `
		FROM licenses
		WHERE key_hash = $1
	`

	row := s.pool.QueryRow(ctx, q, keyHash)
	l, err := scanLicense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

func (s *Store) SetLicenseRevoked(ctx context.Context, id uuid.UUID, revoked bool) (*License, error) {
	const q = `
		UPDATE licenses
		SET revoked = $2
		WHERE id = $1
		RETURNING ` + licenseColumns

	row := s.pool.QueryRow(ctx, q, id, revoked)
	l, err := scanLicense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

func (s *Store) UpdateLicense(ctx context.Context, id uuid.UUID, label string, expiresAt *time.Time) (*License, error) {
	const q = `
		UPDATE licenses
		SET label = $2, expires_at = $3
		WHERE id = $1
		RETURNING ` + licenseColumns

	row := s.pool.QueryRow(ctx, q, id, label, expiresAt)
	l, err := scanLicense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
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
	err := row.Scan(
		&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ExpiresAt, &l.Revoked, &l.CreatedAt,
		&l.LastValidatedAt, &l.ValidationCount,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

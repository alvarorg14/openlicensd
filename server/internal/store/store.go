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
	ID        uuid.UUID
	Label     string
	KeyHash   string
	KeyPrefix string
	ExpiresAt *time.Time
	Revoked   bool
	CreatedAt time.Time
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

func (s *Store) CreateLicense(ctx context.Context, label, keyHash, keyPrefix string, expiresAt *time.Time) (*License, error) {
	const q = `
		INSERT INTO licenses (label, key_hash, key_prefix, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, label, key_hash, key_prefix, expires_at, revoked, created_at
	`

	row := s.pool.QueryRow(ctx, q, label, keyHash, keyPrefix, expiresAt)
	return scanLicense(row)
}

func (s *Store) ListLicenses(ctx context.Context) ([]License, error) {
	const q = `
		SELECT id, label, key_hash, key_prefix, expires_at, revoked, created_at
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
		if err := rows.Scan(&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ExpiresAt, &l.Revoked, &l.CreatedAt); err != nil {
			return nil, err
		}
		licenses = append(licenses, l)
	}

	return licenses, rows.Err()
}

func (s *Store) GetLicenseByKeyHash(ctx context.Context, keyHash string) (*License, error) {
	const q = `
		SELECT id, label, key_hash, key_prefix, expires_at, revoked, created_at
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

func (s *Store) RevokeLicense(ctx context.Context, id uuid.UUID) (*License, error) {
	const q = `
		UPDATE licenses
		SET revoked = TRUE
		WHERE id = $1
		RETURNING id, label, key_hash, key_prefix, expires_at, revoked, created_at
	`

	row := s.pool.QueryRow(ctx, q, id)
	l, err := scanLicense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

func scanLicense(row pgx.Row) (*License, error) {
	var l License
	err := row.Scan(&l.ID, &l.Label, &l.KeyHash, &l.KeyPrefix, &l.ExpiresAt, &l.Revoked, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

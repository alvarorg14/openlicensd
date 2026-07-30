package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ExpirationBasis string

const (
	ExpirationOnCreation        ExpirationBasis = "on_creation"
	ExpirationOnFirstValidation ExpirationBasis = "on_first_validation"
)

type Policy struct {
	ID              uuid.UUID
	ProductID       uuid.UUID
	Name            string
	Description     *string
	DurationDays    *int
	ExpirationBasis ExpirationBasis
	GracePeriodDays int
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const policyColumns = `id, product_id, name, description, duration_days, expiration_basis, grace_period_days, archived_at, created_at, updated_at`

func (s *Store) CreatePolicy(ctx context.Context, productID uuid.UUID, name string, description *string, durationDays *int, expirationBasis ExpirationBasis, gracePeriodDays int) (*Policy, error) {
	const q = `
		INSERT INTO policies (product_id, name, description, duration_days, expiration_basis, grace_period_days)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + policyColumns

	row := s.pool.QueryRow(ctx, q, productID, name, description, durationDays, expirationBasis, gracePeriodDays)
	p, err := scanPolicy(row)
	if err != nil {
		return nil, mapInsertError(err)
	}
	return p, nil
}

func (s *Store) ListPolicies(ctx context.Context, productID *uuid.UUID) ([]Policy, error) {
	var (
		q    string
		args []any
	)

	if productID != nil {
		q = `
			SELECT ` + policyColumns + `
			FROM policies
			WHERE product_id = $1
			ORDER BY created_at DESC
		`
		args = []any{*productID}
	} else {
		q = `
			SELECT ` + policyColumns + `
			FROM policies
			ORDER BY created_at DESC
		`
	}

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPolicies(rows)
}

func (s *Store) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	const q = `
		SELECT ` + policyColumns + `
		FROM policies
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, q, id)
	p, err := scanPolicy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) GetPolicyForProduct(ctx context.Context, id, productID uuid.UUID) (*Policy, error) {
	const q = `
		SELECT ` + policyColumns + `
		FROM policies
		WHERE id = $1 AND product_id = $2
	`

	row := s.pool.QueryRow(ctx, q, id, productID)
	p, err := scanPolicy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdatePolicy(ctx context.Context, id uuid.UUID, name string, description *string, durationDays *int, expirationBasis ExpirationBasis, gracePeriodDays int) (*Policy, error) {
	const q = `
		UPDATE policies
		SET name = $2, description = $3, duration_days = $4, expiration_basis = $5,
		    grace_period_days = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + policyColumns

	row := s.pool.QueryRow(ctx, q, id, name, description, durationDays, expirationBasis, gracePeriodDays)
	p, err := scanPolicy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapInsertError(err)
	}
	return p, nil
}

func (s *Store) DeletePolicy(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM policies WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		if isReferentialViolation(err) {
			return false, ErrConflict
		}
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanPolicy(row pgx.Row) (*Policy, error) {
	var p Policy
	var expirationBasis string
	err := row.Scan(
		&p.ID, &p.ProductID, &p.Name, &p.Description, &p.DurationDays, &expirationBasis,
		&p.GracePeriodDays, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.ExpirationBasis = ExpirationBasis(expirationBasis)
	return &p, nil
}

func scanPolicies(rows pgx.Rows) ([]Policy, error) {
	var policies []Policy
	for rows.Next() {
		var p Policy
		var expirationBasis string
		if err := rows.Scan(
			&p.ID, &p.ProductID, &p.Name, &p.Description, &p.DurationDays, &expirationBasis,
			&p.GracePeriodDays, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.ExpirationBasis = ExpirationBasis(expirationBasis)
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

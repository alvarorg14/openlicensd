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
	ProductName     string
	Name            string
	Description     *string
	DurationDays    *int
	ExpirationBasis ExpirationBasis
	GracePeriodDays int
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const policyColumns = `pol.id, pol.product_id, p.name, pol.name, pol.description, pol.duration_days, pol.expiration_basis, pol.grace_period_days, pol.archived_at, pol.created_at, pol.updated_at`

const policyFromJoin = `
	FROM policies pol
	JOIN products p ON p.id = pol.product_id
`

func (s *Store) CreatePolicy(ctx context.Context, productID uuid.UUID, name string, description *string, durationDays *int, expirationBasis ExpirationBasis, gracePeriodDays int) (*Policy, error) {
	const q = `
		INSERT INTO policies (product_id, name, description, duration_days, expiration_basis, grace_period_days)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, q, productID, name, description, durationDays, expirationBasis, gracePeriodDays).Scan(&id); err != nil {
		return nil, mapInsertError(err)
	}
	return s.GetPolicy(ctx, id)
}

func (s *Store) ListPolicies(ctx context.Context, params PolicyListParams) ([]Policy, int64, error) {
	qb := newQueryBuilder()
	if params.ProductID != nil {
		qb.add("pol.product_id = $%d::uuid", *params.ProductID)
	}
	if params.Search != "" {
		pattern := searchPattern(params.Search)
		qb.add("(pol.name ILIKE $%d OR p.name ILIKE $%d)", pattern, pattern)
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "pol.created_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "pol.id")

	q := `
		SELECT ` + policyColumns + `, COUNT(*) OVER() AS total_count` + policyFromJoin + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanPoliciesWithTotal(rows)
}

func (s *Store) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	const q = `
		SELECT ` + policyColumns + policyFromJoin + `
		WHERE pol.id = $1
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
		SELECT ` + policyColumns + policyFromJoin + `
		WHERE pol.id = $1 AND pol.product_id = $2
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
		RETURNING id
	`

	var returnedID uuid.UUID
	err := s.pool.QueryRow(ctx, q, id, name, description, durationDays, expirationBasis, gracePeriodDays).Scan(&returnedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapInsertError(err)
	}
	return s.GetPolicy(ctx, returnedID)
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
		&p.ID, &p.ProductID, &p.ProductName, &p.Name, &p.Description, &p.DurationDays, &expirationBasis,
		&p.GracePeriodDays, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.ExpirationBasis = ExpirationBasis(expirationBasis)
	return &p, nil
}

func scanPolicies(rows pgx.Rows) ([]Policy, error) {
	policies, _, err := scanPoliciesWithTotal(rows)
	return policies, err
}

func scanPoliciesWithTotal(rows pgx.Rows) ([]Policy, int64, error) {
	var (
		policies   []Policy
		totalCount int64
	)
	for rows.Next() {
		var p Policy
		var expirationBasis string
		if err := rows.Scan(
			&p.ID, &p.ProductID, &p.ProductName, &p.Name, &p.Description, &p.DurationDays, &expirationBasis,
			&p.GracePeriodDays, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		p.ExpirationBasis = ExpirationBasis(expirationBasis)
		policies = append(policies, p)
	}
	return policies, totalCount, rows.Err()
}

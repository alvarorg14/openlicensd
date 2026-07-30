package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Product struct {
	ID          uuid.UUID
	Name        string
	Code        string
	Description *string
	ArchivedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const productColumns = `id, name, code, description, archived_at, created_at, updated_at`

func (s *Store) CreateProduct(ctx context.Context, name, code string, description *string) (*Product, error) {
	const q = `
		INSERT INTO products (name, code, description)
		VALUES ($1, $2, $3)
		RETURNING ` + productColumns

	row := s.pool.QueryRow(ctx, q, name, code, description)
	p, err := scanProduct(row)
	if err != nil {
		return nil, mapInsertError(err)
	}
	return p, nil
}

func (s *Store) ListProducts(ctx context.Context, params ListParams) ([]Product, int64, error) {
	qb := newQueryBuilder()
	if params.Search != "" {
		pattern := searchPattern(params.Search)
		qb.add("(name ILIKE $%d OR code ILIKE $%d)", pattern, pattern)
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "created_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "id")

	q := `
		SELECT ` + productColumns + `, COUNT(*) OVER() AS total_count
		FROM products` + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		products   []Product
		totalCount int64
	)
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Code, &p.Description, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, totalCount, rows.Err()
}

func (s *Store) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
	const q = `
		SELECT ` + productColumns + `
		FROM products
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, q, id)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) GetProductByCode(ctx context.Context, code string) (*Product, error) {
	const q = `
		SELECT ` + productColumns + `
		FROM products
		WHERE code = $1
	`

	row := s.pool.QueryRow(ctx, q, code)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdateProduct(ctx context.Context, id uuid.UUID, name, code string, description *string) (*Product, error) {
	const q = `
		UPDATE products
		SET name = $2, code = $3, description = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + productColumns

	row := s.pool.QueryRow(ctx, q, id, name, code, description)
	p, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) DeleteProduct(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM products WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		if isReferentialViolation(err) {
			return false, ErrConflict
		}
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	err := row.Scan(
		&p.ID, &p.Name, &p.Code, &p.Description, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func isReferentialViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503" || pgErr.Code == "23001"
	}
	msg := err.Error()
	return strings.Contains(msg, "foreign key") || strings.Contains(msg, "violates RESTRICT")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func mapInsertError(err error) error {
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}

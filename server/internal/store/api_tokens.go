package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type APIToken struct {
	ID          uuid.UUID
	Name        string
	TokenHash   string
	TokenPrefix string
	Role        Role
	CreatedBy   *uuid.UUID
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const apiTokenColumns = `
	id, name, token_hash, token_prefix, role, created_by,
	last_used_at, expires_at, revoked_at, created_at, updated_at
`

func (s *Store) CreateAPIToken(
	ctx context.Context,
	name, tokenHash, tokenPrefix string,
	role Role,
	createdBy *uuid.UUID,
	expiresAt *time.Time,
) (*APIToken, error) {
	const q = `
		INSERT INTO api_tokens (name, token_hash, token_prefix, role, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + apiTokenColumns

	row := s.pool.QueryRow(ctx, q, strings.TrimSpace(name), tokenHash, tokenPrefix, role, createdBy, expiresAt)
	tok, err := scanAPIToken(row)
	if err != nil {
		return nil, mapInsertError(err)
	}
	return tok, nil
}

func (s *Store) GetAPITokenByHash(ctx context.Context, tokenHash string) (*APIToken, error) {
	const q = `
		SELECT ` + apiTokenColumns + `
		FROM api_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
	`

	row := s.pool.QueryRow(ctx, q, tokenHash)
	tok, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tok, nil
}

func (s *Store) GetAPITokenByID(ctx context.Context, id uuid.UUID) (*APIToken, error) {
	const q = `
		SELECT ` + apiTokenColumns + `
		FROM api_tokens
		WHERE id = $1
	`

	row := s.pool.QueryRow(ctx, q, id)
	tok, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tok, nil
}

func (s *Store) ListAPITokens(ctx context.Context, params ListParams) ([]APIToken, int64, error) {
	qb := newQueryBuilder()
	if params.Search != "" {
		pattern := searchPattern(params.Search)
		qb.add("(name ILIKE $%d OR token_prefix ILIKE $%d)", pattern, pattern)
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "created_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "id")

	q := `
		SELECT ` + apiTokenColumns + `, COUNT(*) OVER() AS total_count
		FROM api_tokens` + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		tokens     []APIToken
		totalCount int64
	)
	for rows.Next() {
		var tok APIToken
		if err := rows.Scan(
			&tok.ID, &tok.Name, &tok.TokenHash, &tok.TokenPrefix, &tok.Role, &tok.CreatedBy,
			&tok.LastUsedAt, &tok.ExpiresAt, &tok.RevokedAt, &tok.CreatedAt, &tok.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		tokens = append(tokens, tok)
	}

	return tokens, totalCount, rows.Err()
}

func (s *Store) TouchAPIToken(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE api_tokens
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, id)
	return err
}

func (s *Store) RevokeAPIToken(ctx context.Context, id uuid.UUID) (*APIToken, error) {
	const q = `
		UPDATE api_tokens
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING ` + apiTokenColumns

	row := s.pool.QueryRow(ctx, q, id)
	tok, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tok, nil
}

func (s *Store) DeleteAPIToken(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `DELETE FROM api_tokens WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanAPIToken(row pgx.Row) (*APIToken, error) {
	var tok APIToken
	err := row.Scan(
		&tok.ID, &tok.Name, &tok.TokenHash, &tok.TokenPrefix, &tok.Role, &tok.CreatedBy,
		&tok.LastUsedAt, &tok.ExpiresAt, &tok.RevokedAt, &tok.CreatedAt, &tok.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tok, nil
}

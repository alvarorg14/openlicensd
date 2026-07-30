package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TokenHash    string
	AuthProvider string
	UserAgent    *string
	ClientIP     *string
	CreatedAt    time.Time
	LastSeenAt   time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

const sessionColumns = `
	id, user_id, token_hash, auth_provider, user_agent, client_ip,
	created_at, last_seen_at, expires_at, revoked_at
`

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash, authProvider string, userAgent, clientIP *string, expiresAt time.Time) (*Session, error) {
	if authProvider == "" {
		authProvider = AuthProviderLocal
	}

	const q = `
		INSERT INTO sessions (user_id, token_hash, auth_provider, user_agent, client_ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + sessionColumns

	row := s.pool.QueryRow(ctx, q, userID, tokenHash, authProvider, userAgent, clientIP, expiresAt)
	return scanSession(row)
}

func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	const q = `
		SELECT ` + sessionColumns + `
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	row := s.pool.QueryRow(ctx, q, tokenHash)
	sess, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return sess, nil
}

func (s *Store) TouchSession(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	const q = `
		UPDATE sessions
		SET last_seen_at = NOW(), expires_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, id, expiresAt)
	return err
}

func (s *Store) RevokeSession(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, id)
	return err
}

func (s *Store) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	const q = `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, userID)
	return err
}

func (s *Store) RevokeUserSessionsExcept(ctx context.Context, userID, keepID uuid.UUID) error {
	const q = `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`

	_, err := s.pool.Exec(ctx, q, userID, keepID)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	const q = `
		DELETE FROM sessions
		WHERE expires_at < NOW() OR revoked_at IS NOT NULL
	`

	_, err := s.pool.Exec(ctx, q)
	return err
}

func scanSession(row pgx.Row) (*Session, error) {
	var sess Session
	err := row.Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.AuthProvider, &sess.UserAgent, &sess.ClientIP,
		&sess.CreatedAt, &sess.LastSeenAt, &sess.ExpiresAt, &sess.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

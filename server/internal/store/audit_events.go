package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditEvent struct {
	ID             uuid.UUID
	OccurredAt     time.Time
	Action         string
	ResourceType   string
	ResourceID     *uuid.UUID
	ResourceLabel  *string
	ActorUserID    *uuid.UUID
	ActorTokenID   *uuid.UUID
	ActorName      string
	ActorEmail     *string
	ActorRole      string
	AuthMethod     string
	ClientIP       *string
	UserAgent      *string
	RequestID      *string
	RequestMethod  string
	RequestPath    string
	ResponseStatus int
	Metadata       json.RawMessage
}

const auditEventColumns = `
	id, occurred_at, action, resource_type, resource_id, resource_label,
	actor_user_id, actor_token_id, actor_name, actor_email, actor_role,
	auth_method, client_ip, user_agent, request_id, request_method,
	request_path, response_status, metadata
`

func (s *Store) CreateAuditEvent(ctx context.Context, event AuditEvent) error {
	const q = `
		INSERT INTO audit_events (
			action, resource_type, resource_id, resource_label,
			actor_user_id, actor_token_id, actor_name, actor_email, actor_role,
			auth_method, client_ip, user_agent, request_id, request_method,
			request_path, response_status, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := s.pool.Exec(ctx, q,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.ResourceLabel,
		event.ActorUserID,
		event.ActorTokenID,
		event.ActorName,
		event.ActorEmail,
		event.ActorRole,
		event.AuthMethod,
		event.ClientIP,
		event.UserAgent,
		event.RequestID,
		event.RequestMethod,
		event.RequestPath,
		event.ResponseStatus,
		event.Metadata,
	)
	return err
}

func (s *Store) ListAuditEvents(ctx context.Context, params AuditEventListParams) ([]AuditEvent, int64, error) {
	qb := newQueryBuilder()

	if params.Search != "" {
		pattern := searchPattern(params.Search)
		qb.add(`(
			action ILIKE $%d OR resource_type ILIKE $%d OR resource_label ILIKE $%d
			OR actor_name ILIKE $%d OR actor_email ILIKE $%d OR request_path ILIKE $%d
		)`, pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if params.Action != "" {
		qb.add("action = $%d", params.Action)
	}
	if params.ResourceType != "" {
		qb.add("resource_type = $%d", params.ResourceType)
	}
	if params.ActorUserID != nil {
		qb.add("actor_user_id = $%d::uuid", *params.ActorUserID)
	}
	if params.From != nil {
		qb.add("occurred_at >= $%d::timestamptz", *params.From)
	}
	if params.To != nil {
		qb.add("occurred_at <= $%d::timestamptz", *params.To)
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "occurred_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "id")

	q := `
		SELECT ` + auditEventColumns + `, COUNT(*) OVER() AS total_count
		FROM audit_events` + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		events     []AuditEvent
		totalCount int64
	)
	for rows.Next() {
		var ev AuditEvent
		if err := rows.Scan(
			&ev.ID, &ev.OccurredAt, &ev.Action, &ev.ResourceType, &ev.ResourceID, &ev.ResourceLabel,
			&ev.ActorUserID, &ev.ActorTokenID, &ev.ActorName, &ev.ActorEmail, &ev.ActorRole,
			&ev.AuthMethod, &ev.ClientIP, &ev.UserAgent, &ev.RequestID, &ev.RequestMethod,
			&ev.RequestPath, &ev.ResponseStatus, &ev.Metadata,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, ev)
	}

	return events, totalCount, rows.Err()
}

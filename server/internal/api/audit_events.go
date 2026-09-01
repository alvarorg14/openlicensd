package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
)

type auditEventResponse struct {
	ID             uuid.UUID       `json:"id"`
	OccurredAt     string          `json:"occurred_at"`
	Action         string          `json:"action"`
	ResourceType   string          `json:"resource_type"`
	ResourceID     *uuid.UUID      `json:"resource_id,omitempty"`
	ResourceLabel  *string         `json:"resource_label,omitempty"`
	ActorUserID    *uuid.UUID      `json:"actor_user_id,omitempty"`
	ActorTokenID   *uuid.UUID      `json:"actor_token_id,omitempty"`
	ActorName      string          `json:"actor_name"`
	ActorEmail     *string         `json:"actor_email,omitempty"`
	ActorRole      string          `json:"actor_role"`
	AuthMethod     string          `json:"auth_method"`
	ClientIP       *string         `json:"client_ip,omitempty"`
	UserAgent      *string         `json:"user_agent,omitempty"`
	RequestID      *string         `json:"request_id,omitempty"`
	RequestMethod  string          `json:"request_method"`
	RequestPath    string          `json:"request_path"`
	ResponseStatus int             `json:"response_status"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

func auditEventToResponse(ev *store.AuditEvent) auditEventResponse {
	return auditEventResponse{
		ID:             ev.ID,
		OccurredAt:     ev.OccurredAt.Format(timeRFC3339),
		Action:         ev.Action,
		ResourceType:   ev.ResourceType,
		ResourceID:     ev.ResourceID,
		ResourceLabel:  ev.ResourceLabel,
		ActorUserID:    ev.ActorUserID,
		ActorTokenID:   ev.ActorTokenID,
		ActorName:      ev.ActorName,
		ActorEmail:     ev.ActorEmail,
		ActorRole:      ev.ActorRole,
		AuthMethod:     ev.AuthMethod,
		ClientIP:       ev.ClientIP,
		UserAgent:      ev.UserAgent,
		RequestID:      ev.RequestID,
		RequestMethod:  ev.RequestMethod,
		RequestPath:    ev.RequestPath,
		ResponseStatus: ev.ResponseStatus,
		Metadata:       ev.Metadata,
	}
}

func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParamsDefault(r, auditEventSorts, "occurred_at")
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	listParams := store.AuditEventListParams{
		ListParams: store.ListParams{
			Search: params.Search,
			Sort:   params.Sort,
			Order:  params.Order,
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	}

	if action := r.URL.Query().Get("action"); action != "" {
		listParams.Action = action
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		listParams.ResourceType = resourceType
	}
	if raw := r.URL.Query().Get("actor_user_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid actor_user_id")
			return
		}
		listParams.ActorUserID = &raw
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		if _, err := time.Parse(timeRFC3339, raw); err != nil {
			writeError(w, http.StatusBadRequest, "from must be a valid RFC3339 timestamp")
			return
		}
		listParams.From = &raw
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if _, err := time.Parse(timeRFC3339, raw); err != nil {
			writeError(w, http.StatusBadRequest, "to must be a valid RFC3339 timestamp")
			return
		}
		listParams.To = &raw
	}

	events, total, err := s.store.ListAuditEvents(r.Context(), listParams)
	if err != nil {
		writeInternalError(w, r, err, "failed to list audit events")
		return
	}

	resp := make([]auditEventResponse, 0, len(events))
	for i := range events {
		resp = append(resp, auditEventToResponse(&events[i]))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

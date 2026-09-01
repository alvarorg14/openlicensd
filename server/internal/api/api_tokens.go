package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type apiTokenResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	Role        string     `json:"role"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	LastUsedAt  *string    `json:"last_used_at,omitempty"`
	ExpiresAt   *string    `json:"expires_at,omitempty"`
	RevokedAt   *string    `json:"revoked_at,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

type apiTokenWithSecretResponse struct {
	apiTokenResponse
	Token string `json:"token"`
}

type createAPITokenRequest struct {
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	ExpiresAt *string `json:"expires_at"`
}

func apiTokenToResponse(tok *store.APIToken) apiTokenResponse {
	resp := apiTokenResponse{
		ID:          tok.ID,
		Name:        tok.Name,
		TokenPrefix: tok.TokenPrefix,
		Role:        string(tok.Role),
		CreatedBy:   tok.CreatedBy,
		CreatedAt:   tok.CreatedAt.Format(timeRFC3339),
		UpdatedAt:   tok.UpdatedAt.Format(timeRFC3339),
	}
	if tok.LastUsedAt != nil {
		formatted := tok.LastUsedAt.Format(timeRFC3339)
		resp.LastUsedAt = &formatted
	}
	if tok.ExpiresAt != nil {
		formatted := tok.ExpiresAt.Format(timeRFC3339)
		resp.ExpiresAt = &formatted
	}
	if tok.RevokedAt != nil {
		formatted := tok.RevokedAt.Format(timeRFC3339)
		resp.RevokedAt = &formatted
	}
	return resp
}

func requireSessionAuth(w http.ResponseWriter, r *http.Request) (*auth.Principal, bool) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	if principal.AuthMethod == auth.AuthMethodAPIToken {
		writeError(w, http.StatusForbidden, "not available for api token authentication")
		return nil, false
	}
	return principal, true
}

func (s *Server) handleListAPITokens(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSessionAuth(w, r); !ok {
		return
	}

	params, err := parseListParams(r, apiTokenSorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	tokens, total, err := s.store.ListAPITokens(r.Context(), store.ListParams{
		Search: params.Search,
		Sort:   params.Sort,
		Order:  params.Order,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		writeInternalError(w, r, err, "failed to list api tokens")
		return
	}

	resp := make([]apiTokenResponse, 0, len(tokens))
	for i := range tokens {
		resp = append(resp, apiTokenToResponse(&tokens[i]))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

func (s *Server) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	principal, ok := requireSessionAuth(w, r)
	if !ok {
		return
	}

	var req createAPITokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	role, validRole := parseRole(req.Role)
	if !validRole {
		writeError(w, http.StatusBadRequest, "role must be admin, operator, or viewer")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		if strings.TrimSpace(*req.ExpiresAt) == "" {
			writeError(w, http.StatusBadRequest, "expires_at must be omitted or a valid RFC3339 timestamp")
			return
		}
		parsed, err := time.Parse(timeRFC3339, *req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "expires_at must be a valid RFC3339 timestamp")
			return
		}
		if !parsed.After(time.Now()) {
			writeError(w, http.StatusBadRequest, "expires_at must be in the future")
			return
		}
		expiresAt = &parsed
	}

	raw, tokenHash, tokenPrefix, err := auth.GenerateAPIToken()
	if err != nil {
		writeInternalError(w, r, err, "failed to generate api token")
		return
	}

	createdBy := principal.ActingUserID()
	tok, err := s.store.CreateAPIToken(r.Context(), name, tokenHash, tokenPrefix, role, createdBy, expiresAt)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to create api token")
		return
	}

	resp := apiTokenWithSecretResponse{
		apiTokenResponse: apiTokenToResponse(tok),
		Token:            raw,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSessionAuth(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid api token id")
		return
	}

	tok, err := s.store.RevokeAPIToken(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to revoke api token")
		return
	}
	if tok == nil {
		writeError(w, http.StatusNotFound, "api token not found")
		return
	}

	writeJSON(w, http.StatusOK, apiTokenToResponse(tok))
}

func (s *Server) handleDeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSessionAuth(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid api token id")
		return
	}

	deleted, err := s.store.DeleteAPIToken(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to delete api token")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "api token not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

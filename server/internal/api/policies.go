package api

import (
	"encoding/json"
	"net/http"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type policyResponse struct {
	ID              uuid.UUID `json:"id"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	DurationDays    *int      `json:"duration_days"`
	ExpirationBasis string    `json:"expiration_basis"`
	GracePeriodDays int       `json:"grace_period_days"`
	MaxActivations  *int      `json:"max_activations"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

type createPolicyRequest struct {
	ProductID       uuid.UUID `json:"product_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	DurationDays    *int      `json:"duration_days"`
	ExpirationBasis string    `json:"expiration_basis"`
	GracePeriodDays *int      `json:"grace_period_days"`
	MaxActivations  *int      `json:"max_activations"`
}

type updatePolicyRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationDays    *int    `json:"duration_days"`
	ExpirationBasis string  `json:"expiration_basis"`
	GracePeriodDays *int    `json:"grace_period_days"`
	MaxActivations  *int    `json:"max_activations"`
}

func policyToResponse(p *store.Policy) policyResponse {
	return policyResponse{
		ID:              p.ID,
		ProductID:       p.ProductID,
		ProductName:     p.ProductName,
		Name:            p.Name,
		Description:     p.Description,
		DurationDays:    p.DurationDays,
		ExpirationBasis: string(p.ExpirationBasis),
		GracePeriodDays: p.GracePeriodDays,
		MaxActivations:  p.MaxActivations,
		CreatedAt:       p.CreatedAt.Format(timeRFC3339),
		UpdatedAt:       p.UpdatedAt.Format(timeRFC3339),
	}
}

func parseExpirationBasis(value string) (store.ExpirationBasis, bool) {
	switch store.ExpirationBasis(value) {
	case store.ExpirationOnCreation, store.ExpirationOnFirstValidation:
		return store.ExpirationBasis(value), true
	default:
		return "", false
	}
}

func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProductID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	expirationBasis := store.ExpirationOnCreation
	if req.ExpirationBasis != "" {
		var ok bool
		expirationBasis, ok = parseExpirationBasis(req.ExpirationBasis)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid expiration_basis")
			return
		}
	}

	gracePeriodDays := 0
	if req.GracePeriodDays != nil {
		gracePeriodDays = *req.GracePeriodDays
	}

	var maxActivations *int
	if req.MaxActivations != nil {
		var err error
		maxActivations, err = parseMaxActivations(req.MaxActivations)
		if err != nil {
			writeError(w, http.StatusBadRequest, "max_activations must be at least 1")
			return
		}
	}

	policy, err := s.store.CreatePolicy(
		r.Context(),
		req.ProductID,
		req.Name,
		req.Description,
		req.DurationDays,
		expirationBasis,
		gracePeriodDays,
		maxActivations,
	)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to create policy")
		return
	}

	auditResource(r.Context(), policy.ID, policy.Name)

	writeJSON(w, http.StatusCreated, policyToResponse(policy))
}

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r, policySorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	listParams := store.PolicyListParams{
		ListParams: store.ListParams{
			Search: params.Search,
			Sort:   params.Sort,
			Order:  params.Order,
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	}

	if raw := r.URL.Query().Get("product_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		listParams.ProductID = &raw
	}

	policies, total, err := s.store.ListPolicies(r.Context(), listParams)
	if err != nil {
		writeInternalError(w, r, err, "failed to list policies")
		return
	}

	resp := make([]policyResponse, 0, len(policies))
	for _, p := range policies {
		resp = append(resp, policyToResponse(&p))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy id")
		return
	}

	var req updatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	expirationBasis := store.ExpirationOnCreation
	if req.ExpirationBasis != "" {
		var ok bool
		expirationBasis, ok = parseExpirationBasis(req.ExpirationBasis)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid expiration_basis")
			return
		}
	}

	gracePeriodDays := 0
	if req.GracePeriodDays != nil {
		gracePeriodDays = *req.GracePeriodDays
	}

	var maxActivations *int
	if req.MaxActivations != nil {
		var err error
		maxActivations, err = parseMaxActivations(req.MaxActivations)
		if err != nil {
			writeError(w, http.StatusBadRequest, "max_activations must be at least 1")
			return
		}
	}

	policy, err := s.store.UpdatePolicy(
		r.Context(),
		id,
		req.Name,
		req.Description,
		req.DurationDays,
		expirationBasis,
		gracePeriodDays,
		maxActivations,
	)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to update policy")
		return
	}
	if policy == nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	auditResource(r.Context(), policy.ID, policy.Name)

	writeJSON(w, http.StatusOK, policyToResponse(policy))
}

func (s *Server) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid policy id")
		return
	}

	deleted, err := s.store.DeletePolicy(r.Context(), id)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to delete policy")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

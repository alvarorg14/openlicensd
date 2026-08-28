package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/license"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createLicenseRequest struct {
	Label          string     `json:"label"`
	ProductID      uuid.UUID  `json:"product_id"`
	PolicyID       uuid.UUID  `json:"policy_id"`
	ExpiresAt      *time.Time `json:"expires_at"`
	MaxActivations *int       `json:"max_activations"`
}

type updateLicenseRequest struct {
	Label          string     `json:"label"`
	ExpiresAt      *time.Time `json:"expires_at"`
	MaxActivations *int       `json:"max_activations"`
}

type licenseResponse struct {
	ID              uuid.UUID  `json:"id"`
	Label           string     `json:"label"`
	Key             string     `json:"key,omitempty"`
	KeyPrefix       string     `json:"key_prefix"`
	ProductID       uuid.UUID  `json:"product_id"`
	ProductCode     string     `json:"product_code"`
	ProductName     string     `json:"product_name"`
	PolicyID        uuid.UUID  `json:"policy_id"`
	PolicyName      string     `json:"policy_name"`
	ExpiresAt       *time.Time `json:"expires_at"`
	ActivatedAt     *time.Time `json:"activated_at"`
	Revoked         bool       `json:"revoked"`
	CreatedAt       time.Time  `json:"created_at"`
	LastValidatedAt *time.Time `json:"last_validated_at"`
	ValidationCount int64      `json:"validation_count"`
	MaxActivations  *int       `json:"max_activations"`
	ActivationCount int64      `json:"activation_count"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedByName   *string    `json:"created_by_name,omitempty"`
	CreatedByEmail  *string    `json:"created_by_email,omitempty"`
}

func licenseToResponse(lic *store.License, rawKey string) licenseResponse {
	return licenseResponse{
		ID:              lic.ID,
		Label:           lic.Label,
		Key:             rawKey,
		KeyPrefix:       lic.KeyPrefix,
		ProductID:       lic.ProductID,
		ProductCode:     lic.ProductCode,
		ProductName:     lic.ProductName,
		PolicyID:        lic.PolicyID,
		PolicyName:      lic.PolicyName,
		ExpiresAt:       lic.ExpiresAt,
		ActivatedAt:     lic.ActivatedAt,
		Revoked:         lic.Revoked,
		CreatedAt:       lic.CreatedAt,
		LastValidatedAt: lic.LastValidatedAt,
		ValidationCount: lic.ValidationCount,
		MaxActivations:  lic.MaxActivations,
		ActivationCount: lic.ActivationCount,
		CreatedBy:       lic.CreatedBy,
		CreatedByName:   lic.CreatedByName,
		CreatedByEmail:  lic.CreatedByEmail,
	}
}

func resolveInitialExpiry(policy *store.Policy, override *time.Time, now time.Time) *time.Time {
	if override != nil {
		return override
	}
	if policy.DurationDays == nil {
		return nil
	}
	if policy.ExpirationBasis == store.ExpirationOnFirstValidation {
		return nil
	}
	return license.ComputeExpiry(policy.DurationDays, now)
}

func (s *Server) handleCreateLicense(w http.ResponseWriter, r *http.Request) {
	var req createLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	if req.ProductID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}
	if req.PolicyID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	policy, err := s.store.GetPolicyForProduct(r.Context(), req.PolicyID, req.ProductID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load policy")
		return
	}
	if policy == nil {
		writeError(w, http.StatusBadRequest, "policy does not belong to product")
		return
	}

	rawKey, keyHash, keyPrefix, err := license.GenerateKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate license key")
		return
	}

	expiresAt := resolveInitialExpiry(policy, req.ExpiresAt, time.Now())

	var maxActivations *int
	if req.MaxActivations != nil {
		var err error
		maxActivations, err = parseMaxActivations(req.MaxActivations)
		if err != nil {
			writeError(w, http.StatusBadRequest, "max_activations must be at least 1")
			return
		}
	} else {
		maxActivations = policy.MaxActivations
	}

	var createdBy *uuid.UUID
	if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
		createdBy = &principal.UserID
	}

	lic, err := s.store.CreateLicense(r.Context(), req.Label, keyHash, keyPrefix, req.ProductID, req.PolicyID, expiresAt, maxActivations, createdBy)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create license")
		return
	}

	writeJSON(w, http.StatusCreated, licenseToResponse(lic, rawKey))
}

func (s *Server) handleListLicenses(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r, licenseSorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	listParams := store.LicenseListParams{
		ListParams: store.ListParams{
			Search: params.Search,
			Sort:   params.Sort,
			Order:  params.Order,
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	}

	if status := r.URL.Query().Get("status"); status != "" {
		switch status {
		case "active", "expired", "revoked":
			listParams.Status = status
		default:
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}

	if raw := r.URL.Query().Get("product_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		listParams.ProductID = &raw
	}

	if raw := r.URL.Query().Get("policy_id"); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid policy_id")
			return
		}
		listParams.PolicyID = &raw
	}

	licenses, total, err := s.store.ListLicenses(r.Context(), listParams)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list licenses")
		return
	}

	resp := make([]licenseResponse, 0, len(licenses))
	for _, lic := range licenses {
		resp = append(resp, licenseToResponse(&lic, ""))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

type licenseStatsResponse struct {
	Total   int64 `json:"total"`
	Active  int64 `json:"active"`
	Expired int64 `json:"expired"`
	Revoked int64 `json:"revoked"`
}

func (s *Server) handleLicenseStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.LicenseStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load license stats")
		return
	}

	writeJSON(w, http.StatusOK, licenseStatsResponse{
		Total:   stats.Total,
		Active:  stats.Active,
		Expired: stats.Expired,
		Revoked: stats.Revoked,
	})
}

func (s *Server) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.GetLicenseByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseToResponse(lic, ""))
}

func (s *Server) handleUpdateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	var req updateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required")
		return
	}

	if req.MaxActivations != nil {
		if _, err := parseMaxActivations(req.MaxActivations); err != nil {
			writeError(w, http.StatusBadRequest, "max_activations must be at least 1")
			return
		}
	}

	lic, err := s.store.UpdateLicense(r.Context(), id, req.Label, req.ExpiresAt, req.MaxActivations)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseToResponse(lic, ""))
}

func (s *Server) handleDeleteLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	deleted, err := s.store.DeleteLicense(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete license")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRevokeLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.SetLicenseRevoked(r.Context(), id, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseToResponse(lic, ""))
}

func (s *Server) handleUnrevokeLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.SetLicenseRevoked(r.Context(), id, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unrevoke license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseToResponse(lic, ""))
}

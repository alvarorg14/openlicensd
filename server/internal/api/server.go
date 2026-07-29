package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/harbor"
	"github.com/openlicensd/openlicensd/server/internal/license"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

type Server struct {
	cfg    *config.Config
	auth   *auth.Service
	store  *store.Store
	harbor *harbor.Client
}

func New(cfg *config.Config, st *store.Store) *Server {
	srv := &Server{
		cfg:   cfg,
		auth:  auth.NewService(cfg.AdminUser, cfg.AdminPasswordHash, cfg.JWTSecret),
		store: st,
	}

	if cfg.Harbor.Enabled {
		client, err := harbor.New(
			cfg.Harbor.URL,
			cfg.Harbor.AdminUsername,
			cfg.Harbor.AdminPassword,
			cfg.Harbor.InsecureSkipVerify,
			cfg.Harbor.Debug,
		)
		if err != nil {
			log.Fatalf("harbor client: %v", err)
		}
		srv.harbor = client
	}

	return srv
}

func (s *Server) Router(staticHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/validate", s.handleValidate)
		if s.cfg.Harbor.Enabled {
			r.Post("/registry-credentials", s.handleRegistryCredentials)
		}

		r.Group(func(r chi.Router) {
			r.Use(s.auth.Middleware)
			r.Post("/licenses", s.handleCreateLicense)
			r.Get("/licenses", s.handleListLicenses)
			r.Patch("/licenses/{id}", s.handleUpdateLicense)
			r.Delete("/licenses/{id}", s.handleDeleteLicense)
			r.Patch("/licenses/{id}/revoke", s.handleRevokeLicense)
			r.Patch("/licenses/{id}/activate", s.handleActivateLicense)
		})
	})

	if staticHandler != nil {
		r.NotFound(staticHandler.ServeHTTP)
		r.MethodNotAllowed(staticHandler.ServeHTTP)
	}

	return r
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := s.auth.Login(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

type createLicenseRequest struct {
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type updateLicenseRequest struct {
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type licenseResponse struct {
	ID               uuid.UUID  `json:"id"`
	Label            string     `json:"label"`
	Key              string     `json:"key,omitempty"`
	KeyPrefix        string     `json:"key_prefix"`
	ExpiresAt        *time.Time `json:"expires_at"`
	Revoked          bool       `json:"revoked"`
	CreatedAt        time.Time  `json:"created_at"`
	LastValidatedAt  *time.Time `json:"last_validated_at"`
	ValidationCount  int64      `json:"validation_count"`
}

func licenseToResponse(lic *store.License, rawKey string) licenseResponse {
	return licenseResponse{
		ID:              lic.ID,
		Label:           lic.Label,
		Key:             rawKey,
		KeyPrefix:       lic.KeyPrefix,
		ExpiresAt:       lic.ExpiresAt,
		Revoked:         lic.Revoked,
		CreatedAt:       lic.CreatedAt,
		LastValidatedAt: lic.LastValidatedAt,
		ValidationCount: lic.ValidationCount,
	}
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

	rawKey, keyHash, keyPrefix, err := license.GenerateKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate license key")
		return
	}

	lic, err := s.store.CreateLicense(r.Context(), req.Label, keyHash, keyPrefix, req.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create license")
		return
	}

	writeJSON(w, http.StatusCreated, licenseToResponse(lic, rawKey))
}

func (s *Server) handleListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := s.store.ListLicenses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list licenses")
		return
	}

	resp := make([]licenseResponse, 0, len(licenses))
	for _, lic := range licenses {
		resp = append(resp, licenseToResponse(&lic, ""))
	}

	writeJSON(w, http.StatusOK, resp)
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

	lic, err := s.store.UpdateLicense(r.Context(), id, req.Label, req.ExpiresAt)
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

func (s *Server) handleActivateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.SetLicenseRevoked(r.Context(), id, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to activate license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseToResponse(lic, ""))
}

type validateRequest struct {
	Key string `json:"key"`
}

type registryCredentialsResponse struct {
	Registry  string `json:"registry"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	ExpiresAt int64  `json:"expires_at"`
}

func (s *Server) resolveValidLicense(ctx context.Context, rawKey string) (*store.License, license.ValidationResult, error) {
	keyHash := license.HashKey(rawKey)
	lic, err := s.store.GetLicenseByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, license.ValidationResult{}, err
	}

	if lic == nil {
		return nil, license.ValidationResult{Valid: false, Reason: "not_found"}, nil
	}

	_ = s.store.RecordValidation(ctx, lic.ID)

	result := license.Validate(lic.ExpiresAt, lic.Revoked, time.Now())
	if !result.Valid {
		return lic, result, nil
	}

	return lic, result, nil
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	_, result, err := s.resolveValidLicense(r.Context(), req.Key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate license")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRegistryCredentials(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	lic, result, err := s.resolveValidLicense(r.Context(), req.Key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate license")
		return
	}

	if !result.Valid {
		reason := result.Reason
		if reason == "" {
			reason = "invalid"
		}
		writeError(w, http.StatusForbidden, reason)
		return
	}

	creds, err := s.harbor.CreateEphemeralRobot(
		r.Context(),
		s.cfg.Harbor.Projects,
		s.cfg.Harbor.RobotDurationDays,
		s.cfg.Harbor.RobotNamePrefix,
		lic.KeyPrefix,
	)
	if err != nil {
		log.Printf("registry credentials: harbor create robot failed: %v", err)
		message := "failed to issue registry credentials"
		if s.cfg.Harbor.Debug {
			message = fmt.Sprintf("%s: %v", message, err)
		}
		writeError(w, http.StatusBadGateway, message)
		return
	}

	if err := s.harbor.CleanupExpiredRobots(r.Context(), s.cfg.Harbor.RobotNamePrefix); err != nil {
		log.Printf("registry credentials: harbor cleanup failed: %v", err)
	}

	writeJSON(w, http.StatusOK, registryCredentialsResponse{
		Registry:  s.harbor.RegistryHost(),
		Username:  creds.Name,
		Secret:    creds.Secret,
		ExpiresAt: creds.ExpiresAt,
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

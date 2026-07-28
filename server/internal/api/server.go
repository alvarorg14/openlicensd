package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/license"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

type Server struct {
	cfg  *config.Config
	auth *auth.Service
	store *store.Store
}

func New(cfg *config.Config, st *store.Store) *Server {
	return &Server{
		cfg:   cfg,
		auth:  auth.NewService(cfg.AdminUser, cfg.AdminPasswordHash, cfg.JWTSecret),
		store: st,
	}
}

func (s *Server) Router(staticHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/validate", s.handleValidate)

		r.Group(func(r chi.Router) {
			r.Use(s.auth.Middleware)
			r.Post("/licenses", s.handleCreateLicense)
			r.Get("/licenses", s.handleListLicenses)
			r.Patch("/licenses/{id}/revoke", s.handleRevokeLicense)
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

type licenseResponse struct {
	ID        uuid.UUID  `json:"id"`
	Label     string     `json:"label"`
	Key       string     `json:"key,omitempty"`
	KeyPrefix string     `json:"key_prefix"`
	ExpiresAt *time.Time `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
	CreatedAt time.Time  `json:"created_at"`
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

	writeJSON(w, http.StatusCreated, licenseResponse{
		ID:        lic.ID,
		Label:     lic.Label,
		Key:       rawKey,
		KeyPrefix: lic.KeyPrefix,
		ExpiresAt: lic.ExpiresAt,
		Revoked:   lic.Revoked,
		CreatedAt: lic.CreatedAt,
	})
}

func (s *Server) handleListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := s.store.ListLicenses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list licenses")
		return
	}

	resp := make([]licenseResponse, 0, len(licenses))
	for _, lic := range licenses {
		resp = append(resp, licenseResponse{
			ID:        lic.ID,
			Label:     lic.Label,
			KeyPrefix: lic.KeyPrefix,
			ExpiresAt: lic.ExpiresAt,
			Revoked:   lic.Revoked,
			CreatedAt: lic.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRevokeLicense(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license id")
		return
	}

	lic, err := s.store.RevokeLicense(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke license")
		return
	}
	if lic == nil {
		writeError(w, http.StatusNotFound, "license not found")
		return
	}

	writeJSON(w, http.StatusOK, licenseResponse{
		ID:        lic.ID,
		Label:     lic.Label,
		KeyPrefix: lic.KeyPrefix,
		ExpiresAt: lic.ExpiresAt,
		Revoked:   lic.Revoked,
		CreatedAt: lic.CreatedAt,
	})
}

type validateRequest struct {
	Key string `json:"key"`
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

	keyHash := license.HashKey(req.Key)
	lic, err := s.store.GetLicenseByKeyHash(r.Context(), keyHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate license")
		return
	}

	if lic == nil {
		writeJSON(w, http.StatusOK, license.ValidationResult{Valid: false, Reason: "not_found"})
		return
	}

	result := license.Validate(lic.ExpiresAt, lic.Revoked, time.Now())
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

var ErrNotFound = errors.New("not found")

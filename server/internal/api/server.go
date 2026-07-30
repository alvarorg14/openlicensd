package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/config"
	"github.com/openlicensd/openlicensd/server/internal/harbor"
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

			r.Post("/products", s.handleCreateProduct)
			r.Get("/products", s.handleListProducts)
			r.Patch("/products/{id}", s.handleUpdateProduct)
			r.Delete("/products/{id}", s.handleDeleteProduct)

			r.Post("/policies", s.handleCreatePolicy)
			r.Get("/policies", s.handleListPolicies)
			r.Patch("/policies/{id}", s.handleUpdatePolicy)
			r.Delete("/policies/{id}", s.handleDeletePolicy)
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

func writeStoreError(w http.ResponseWriter, err error, notFoundMessage string) bool {
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "resource is referenced by other records")
		return true
	}
	return false
}

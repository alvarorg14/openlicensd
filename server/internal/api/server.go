package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/clientip"
	"github.com/alvarorg14/openlicensd/server/internal/config"
	"github.com/alvarorg14/openlicensd/server/internal/harbor"
	appoidc "github.com/alvarorg14/openlicensd/server/internal/oidc"
	"github.com/alvarorg14/openlicensd/server/internal/ratelimit"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg      *config.Config
	auth     *auth.Service
	store    *store.Store
	harbor   *harbor.Client
	oidc     *appoidc.Client
	limiter  *ratelimit.Limiter
	clientIP *clientip.Resolver
}

func New(ctx context.Context, cfg *config.Config, st *store.Store) *Server {
	sessionTTL := time.Duration(cfg.SessionTTLHours) * time.Hour
	clientIP, err := clientip.NewResolver(cfg.TrustedProxies)
	if err != nil {
		log.Fatalf("client ip resolver: %v", err)
	}

	srv := &Server{
		cfg:      cfg,
		auth:     auth.NewService(st, sessionTTL, cfg.CookieSecure),
		store:    st,
		limiter:  ratelimit.New(cfg.RateLimit),
		clientIP: clientIP,
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

	if cfg.OIDC.Enabled {
		client, err := appoidc.New(ctx, appoidc.Config{
			IssuerURL:    cfg.OIDC.IssuerURL,
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  cfg.OIDC.RedirectURL,
			Scopes:       cfg.OIDC.Scopes,
		})
		if err != nil {
			log.Fatalf("oidc client: %v", err)
		}
		srv.oidc = client
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
		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit(ratelimit.ScopeLogin))
			r.Post("/auth/login", s.handleLogin)
			if s.cfg.OIDC.Enabled {
				r.Get("/auth/oidc/login", s.handleOIDCLogin)
				r.Get("/auth/oidc/callback", s.handleOIDCCallback)
			}
		})
		r.Get("/auth/providers", s.handleAuthProviders)
		r.Group(func(r chi.Router) {
			r.Use(s.rateLimit(ratelimit.ScopePublic))
			r.Post("/validate", s.handleValidate)
			if s.cfg.Harbor.Enabled {
				r.Post("/registry-credentials", s.handleRegistryCredentials)
			}
		})

		r.Group(func(r chi.Router) {
			r.Use(s.auth.Middleware)

			r.Get("/auth/me", s.handleMe)
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/password", s.handleChangeOwnPassword)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(store.RoleViewer, store.RoleOperator, store.RoleAdmin))

				r.Get("/licenses/stats", s.handleLicenseStats)
				r.Get("/licenses", s.handleListLicenses)
				r.Get("/licenses/{id}/machines", s.handleListLicenseMachines)
				r.Get("/products", s.handleListProducts)
				r.Get("/policies", s.handleListPolicies)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(store.RoleOperator, store.RoleAdmin))

				r.Post("/licenses", s.handleCreateLicense)
				r.Patch("/licenses/{id}", s.handleUpdateLicense)
				r.Delete("/licenses/{id}", s.handleDeleteLicense)
				r.Patch("/licenses/{id}/revoke", s.handleRevokeLicense)
				r.Patch("/licenses/{id}/activate", s.handleActivateLicense)
				r.Patch("/licenses/{id}/machines/{machineId}", s.handleUpdateLicenseMachine)
				r.Delete("/licenses/{id}/machines/{machineId}", s.handleReleaseLicenseMachine)

				r.Post("/products", s.handleCreateProduct)
				r.Patch("/products/{id}", s.handleUpdateProduct)
				r.Delete("/products/{id}", s.handleDeleteProduct)

				r.Post("/policies", s.handleCreatePolicy)
				r.Patch("/policies/{id}", s.handleUpdatePolicy)
				r.Delete("/policies/{id}", s.handleDeletePolicy)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(store.RoleAdmin))

				r.Post("/users", s.handleCreateUser)
				r.Get("/users", s.handleListUsers)
				r.Patch("/users/{id}", s.handleUpdateUser)
				r.Patch("/users/{id}/password", s.handleSetUserPassword)
				r.Patch("/users/{id}/disable", s.handleDisableUser)
				r.Patch("/users/{id}/enable", s.handleEnableUser)
				r.Delete("/users/{id}", s.handleDeleteUser)
			})
		})
	})

	if staticHandler != nil {
		r.NotFound(staticHandler.ServeHTTP)
		r.MethodNotAllowed(staticHandler.ServeHTTP)
	}

	return r
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User map[string]any `json:"user"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.LocalLoginEnabled {
		writeError(w, http.StatusForbidden, "local login is disabled")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userAgent := r.UserAgent()
	clientIP := s.clientIP.From(r)

	user, tokens, err := s.auth.Login(r.Context(), req.Email, req.Password, userAgent, clientIP)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	s.auth.SetSessionCookies(w, tokens)

	writeJSON(w, http.StatusOK, loginResponse{
		User: map[string]any{
			"id":            user.ID,
			"email":         user.Email,
			"name":          user.Name,
			"role":          user.Role,
			"auth_provider": user.AuthProvider,
			"has_password":  user.PasswordHash != nil,
		},
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

func writeStoreError(w http.ResponseWriter, err error, notFoundMessage string) bool {
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "resource is referenced by other records")
		return true
	}
	return false
}

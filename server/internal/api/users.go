package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openlicensd/openlicensd/server/internal/auth"
	"github.com/openlicensd/openlicensd/server/internal/store"
)

type userResponse struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	AuthProvider string     `json:"auth_provider"`
	DisabledAt   *string    `json:"disabled_at,omitempty"`
	LastLoginAt  *string    `json:"last_login_at,omitempty"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type setPasswordRequest struct {
	Password string `json:"password"`
}

func userToResponse(u *store.User) userResponse {
	resp := userResponse{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		Role:         string(u.Role),
		AuthProvider: u.AuthProvider,
		CreatedAt:    u.CreatedAt.Format(timeRFC3339),
		UpdatedAt:    u.UpdatedAt.Format(timeRFC3339),
	}
	if u.DisabledAt != nil {
		formatted := u.DisabledAt.Format(timeRFC3339)
		resp.DisabledAt = &formatted
	}
	if u.LastLoginAt != nil {
		formatted := u.LastLoginAt.Format(timeRFC3339)
		resp.LastLoginAt = &formatted
	}
	return resp
}

func parseRole(role string) (store.Role, bool) {
	switch store.Role(role) {
	case store.RoleAdmin, store.RoleOperator, store.RoleViewer:
		return store.Role(role), true
	default:
		return "", false
	}
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userToResponse(&u))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	role, ok := parseRole(req.Role)
	if !ok {
		writeError(w, http.StatusBadRequest, "role must be admin, operator, or viewer")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, strings.TrimSpace(req.Name), &hash, role, store.AuthProviderLocal, nil)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, userToResponse(user))
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	role, ok := parseRole(req.Role)
	if !ok {
		writeError(w, http.StatusBadRequest, "role must be admin, operator, or viewer")
		return
	}

	user, err := s.store.UpdateUser(r.Context(), id, req.Email, strings.TrimSpace(req.Name), role)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (s *Server) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req setPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := s.store.SetUserPassword(r.Context(), id, hash); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDisableUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	principal, ok := auth.PrincipalFromContext(r.Context())
	if ok && principal.UserID == id {
		writeError(w, http.StatusBadRequest, "cannot disable your own account")
		return
	}

	user, err := s.store.SetUserDisabled(r.Context(), id, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to disable user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	_ = s.store.RevokeAllUserSessions(r.Context(), id)

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (s *Server) handleEnableUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := s.store.SetUserDisabled(r.Context(), id, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enable user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	principal, ok := auth.PrincipalFromContext(r.Context())
	if ok && principal.UserID == id {
		writeError(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	deleted, err := s.store.DeleteUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":            principal.UserID,
		"email":         principal.Email,
		"name":          principal.Name,
		"role":          principal.Role,
		"auth_provider": principal.AuthProvider,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if ok {
		_ = s.auth.Logout(r.Context(), principal.SessionID)
	}
	s.auth.ClearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := s.store.SetUserPassword(r.Context(), principal.UserID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuthProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"local": true,
		"oidc":  false,
	})
}

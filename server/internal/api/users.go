package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/alvarorg14/openlicensd/server/internal/version"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type userResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	AuthProvider string    `json:"auth_provider"`
	DisabledAt   *string   `json:"disabled_at,omitempty"`
	LastLoginAt  *string   `json:"last_login_at,omitempty"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
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

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	Password        string `json:"password"`
}

type sessionMeResponse struct {
	ID            uuid.UUID       `json:"id"`
	Email         string          `json:"email"`
	Name          string          `json:"name"`
	Role          store.Role      `json:"role"`
	AuthProvider  string          `json:"auth_provider"`
	AuthMethod    auth.AuthMethod `json:"auth_method"`
	HasPassword   bool            `json:"has_password"`
	PictureURL    *string         `json:"picture_url"`
	ServerVersion string          `json:"server_version"`
}

type apiTokenMeResponse struct {
	AuthMethod    auth.AuthMethod `json:"auth_method"`
	Name          string          `json:"name"`
	Role          store.Role      `json:"role"`
	TokenID       uuid.UUID       `json:"token_id"`
	ServerVersion string          `json:"server_version"`
}

const minPasswordLength = 8

func validatePassword(password string) string {
	if password == "" {
		return "password is required"
	}
	if len(password) < minPasswordLength {
		return "password must be at least 8 characters"
	}
	return ""
}

func userToResponse(u *store.User) userResponse {
	resp := userResponse{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		Role:         string(u.Role),
		AuthProvider: u.AuthProvider,
		CreatedAt:    formatRFC3339(u.CreatedAt),
		UpdatedAt:    formatRFC3339(u.UpdatedAt),
	}
	if u.DisabledAt != nil {
		resp.DisabledAt = formatRFC3339Ptr(u.DisabledAt)
	}
	if u.LastLoginAt != nil {
		resp.LastLoginAt = formatRFC3339Ptr(u.LastLoginAt)
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

// rejectIfLastAdmin writes a 400 and returns true when mutating user would
// leave the server with zero enabled admins.
func (s *Server) rejectIfLastAdmin(w http.ResponseWriter, r *http.Request, user *store.User, message string) bool {
	if user.Role != store.RoleAdmin || user.DisabledAt != nil {
		return false
	}
	count, err := s.store.CountAdmins(r.Context())
	if err != nil {
		writeInternalError(w, r, err, "failed to count admins")
		return true
	}
	if count <= 1 {
		writeError(w, http.StatusBadRequest, message)
		return true
	}
	return false
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r, userSorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	users, total, err := s.store.ListUsers(r.Context(), store.ListParams{
		Search: params.Search,
		Sort:   params.Sort,
		Order:  params.Order,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		writeInternalError(w, r, err, "failed to list users")
		return
	}

	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userToResponse(&u))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to load user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
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
	if msg := validatePassword(req.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	role, ok := parseRole(req.Role)
	if !ok {
		writeError(w, http.StatusBadRequest, "role must be admin, operator, or viewer")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeInternalError(w, r, err, "failed to hash password")
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, strings.TrimSpace(req.Name), &hash, role, store.AuthProviderLocal, nil)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to create user")
		return
	}

	auditResource(r.Context(), user.ID, user.Name)

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

	if role != store.RoleAdmin {
		existing, loadErr := s.store.GetUserByID(r.Context(), id)
		if loadErr != nil {
			writeInternalError(w, r, loadErr, "failed to load user")
			return
		}
		if existing == nil {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if s.rejectIfLastAdmin(w, r, existing, "cannot demote the last admin") {
			return
		}
	}

	user, err := s.store.UpdateUser(r.Context(), id, req.Email, strings.TrimSpace(req.Name), role)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to update user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	auditResource(r.Context(), user.ID, user.Name)

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
	if msg := validatePassword(req.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeInternalError(w, r, err, "failed to hash password")
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

	existing, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to load user")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if s.rejectIfLastAdmin(w, r, existing, "cannot disable the last admin") {
		return
	}

	user, err := s.store.SetUserDisabled(r.Context(), id, true)
	if err != nil {
		writeInternalError(w, r, err, "failed to disable user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err := s.store.RevokeAllUserSessions(r.Context(), id); err != nil {
		logging.FromContext(r.Context()).Warn("revoke all user sessions failed",
			slog.String("user_id", id.String()),
			slog.Any("err", err),
		)
	}

	auditResource(r.Context(), user.ID, user.Name)

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
		writeInternalError(w, r, err, "failed to enable user")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	auditResource(r.Context(), user.ID, user.Name)

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

	existing, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to load user")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if s.rejectIfLastAdmin(w, r, existing, "cannot delete the last admin") {
		return
	}

	deleted, err := s.store.DeleteUser(r.Context(), id)
	if err != nil {
		writeInternalError(w, r, err, "failed to delete user")
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

	if principal.AuthMethod == auth.AuthMethodAPIToken {
		writeJSON(w, http.StatusOK, apiTokenMeResponse{
			AuthMethod:    auth.AuthMethodAPIToken,
			Name:          principal.Name,
			Role:          principal.Role,
			TokenID:       principal.TokenID,
			ServerVersion: version.Version,
		})
		return
	}

	writeJSON(w, http.StatusOK, sessionMeResponse{
		ID:            principal.UserID,
		Email:         principal.Email,
		Name:          principal.Name,
		Role:          principal.Role,
		AuthProvider:  principal.AuthProvider,
		AuthMethod:    auth.AuthMethodSession,
		HasPassword:   principal.HasPassword,
		PictureURL:    principal.PictureURL,
		ServerVersion: version.Version,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if principal.AuthMethod == auth.AuthMethodAPIToken {
		writeError(w, http.StatusForbidden, "not available for api token authentication")
		return
	}

	if err := s.auth.Logout(r.Context(), principal.SessionID); err != nil {
		logging.FromContext(r.Context()).Warn("logout failed",
			slog.String("session_id", principal.SessionID.String()),
			slog.Any("err", err),
		)
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
	if principal.AuthMethod == auth.AuthMethodAPIToken {
		writeError(w, http.StatusForbidden, "not available for api token authentication")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CurrentPassword == "" {
		writeError(w, http.StatusBadRequest, "current password is required")
		return
	}
	if msg := validatePassword(req.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), principal.UserID)
	if err != nil {
		writeInternalError(w, r, err, "failed to load user")
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if user.PasswordHash == nil {
		writeError(w, http.StatusBadRequest, "password change is not available for this account")
		return
	}
	if !auth.VerifyPassword(*user.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusBadRequest, "current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeInternalError(w, r, err, "failed to hash password")
		return
	}

	if err := s.store.SetUserPassword(r.Context(), principal.UserID, hash); err != nil {
		writeInternalError(w, r, err, "failed to set password")
		return
	}

	if err := s.store.RevokeUserSessionsExcept(r.Context(), principal.UserID, principal.SessionID); err != nil {
		writeInternalError(w, r, err, "failed to revoke sessions")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuthProviders(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"local": s.cfg.LocalLoginEnabled,
		"oidc":  s.cfg.OIDC.Enabled,
	}
	if s.cfg.OIDC.Enabled {
		resp["oidc_name"] = s.cfg.OIDC.ProviderName
		resp["oidc_login_url"] = "/api/v1/auth/oidc/login"
	}
	writeJSON(w, http.StatusOK, resp)
}

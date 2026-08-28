package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/version"
)

func TestRolePermissions(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	viewerEmail := fmt.Sprintf("viewer-%d@example.com", time.Now().UnixNano())
	createViewer := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    viewerEmail,
		"name":     "Viewer User",
		"password": "viewer-password",
		"role":     "viewer",
	}, adminCookies)
	if createViewer.Code != http.StatusCreated {
		t.Fatalf("create viewer status=%d body=%s", createViewer.Code, createViewer.Body.String())
	}

	viewerCookies := login(t, handler, viewerEmail, "viewer-password")

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/licenses", nil, viewerCookies)
	if listResp.Code != http.StatusOK {
		t.Fatalf("viewer list licenses status=%d", listResp.Code)
	}

	writeResp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Forbidden Product",
		"code": "forbidden-product",
	}, viewerCookies)
	if writeResp.Code != http.StatusForbidden {
		t.Fatalf("viewer create product status=%d want 403", writeResp.Code)
	}

	usersResp := doJSON(t, handler, http.MethodGet, "/api/v1/users", nil, viewerCookies)
	if usersResp.Code != http.StatusForbidden {
		t.Fatalf("viewer list users status=%d want 403", usersResp.Code)
	}
}

func TestCSRFRequired(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	sessionOnly := []*http.Cookie{findCookie(cookies, auth.SessionCookieName)}
	resp := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "CSRF Test",
		"code": "csrf-test",
	}, sessionOnly)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing csrf status=%d want 401", resp.Code)
	}
}

func TestAuthProviders(t *testing.T) {
	env := setupTestEnv(t)

	resp := doJSON(t, env.Handler, http.MethodGet, "/api/v1/auth/providers", nil, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("providers status=%d", resp.Code)
	}
}

func TestChangeOwnPassword(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler

	cookies := login(t, handler, env.Email, env.Password)
	otherCookies := login(t, handler, env.Email, env.Password)

	wrongResp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/password", map[string]string{
		"current_password": env.Password,
		"password":         "short",
	}, cookies)
	if wrongResp.Code != http.StatusBadRequest {
		t.Fatalf("short password status=%d want 400 body=%s", wrongResp.Code, wrongResp.Body.String())
	}

	wrongResp = doJSON(t, handler, http.MethodPost, "/api/v1/auth/password", map[string]string{
		"current_password": "wrong-password",
		"password":         "new-password-123",
	}, cookies)
	if wrongResp.Code != http.StatusBadRequest {
		t.Fatalf("wrong current password status=%d want 400 body=%s", wrongResp.Code, wrongResp.Body.String())
	}

	newPassword := "new-password-123"
	changeResp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/password", map[string]string{
		"current_password": env.Password,
		"password":         newPassword,
	}, cookies)
	if changeResp.Code != http.StatusNoContent {
		t.Fatalf("change password status=%d want 204 body=%s", changeResp.Code, changeResp.Body.String())
	}

	meResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, cookies)
	if meResp.Code != http.StatusOK {
		t.Fatalf("current session after password change status=%d want 200", meResp.Code)
	}

	otherMeResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, otherCookies)
	if otherMeResp.Code != http.StatusUnauthorized {
		t.Fatalf("other session after password change status=%d want 401", otherMeResp.Code)
	}

	oldLoginResp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    env.Email,
		"password": env.Password,
	}, nil)
	if oldLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("old password login status=%d want 401", oldLoginResp.Code)
	}

	newCookies := login(t, handler, env.Email, newPassword)
	newMeResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, newCookies)
	if newMeResp.Code != http.StatusOK {
		t.Fatalf("new password login status=%d want 200", newMeResp.Code)
	}
}

func TestGetCurrentUserIncludesServerVersion(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	cookies := login(t, handler, env.Email, env.Password)

	meResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, cookies)
	if meResp.Code != http.StatusOK {
		t.Fatalf("auth/me status=%d want 200 body=%s", meResp.Code, meResp.Body.String())
	}

	var me map[string]any
	if err := json.Unmarshal(meResp.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode auth/me: %v", err)
	}

	got, ok := me["server_version"].(string)
	if !ok || got == "" {
		t.Fatalf("server_version missing or empty: %#v", me["server_version"])
	}
	if got != version.Version {
		t.Fatalf("server_version=%q want %q", got, version.Version)
	}
}

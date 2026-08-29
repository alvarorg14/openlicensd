package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCreateUserPasswordValidation(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	email := fmt.Sprintf("empty-pw-%d@example.com", time.Now().UnixNano())
	emptyResp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    email,
		"name":     "Empty Password",
		"password": "",
		"role":     "viewer",
	}, adminCookies)
	if emptyResp.Code != http.StatusBadRequest {
		t.Fatalf("empty password status=%d want 400 body=%s", emptyResp.Code, emptyResp.Body.String())
	}

	shortEmail := fmt.Sprintf("short-pw-%d@example.com", time.Now().UnixNano())
	shortResp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    shortEmail,
		"name":     "Short Password",
		"password": "short",
		"role":     "viewer",
	}, adminCookies)
	if shortResp.Code != http.StatusBadRequest {
		t.Fatalf("short password status=%d want 400 body=%s", shortResp.Code, shortResp.Body.String())
	}

	okEmail := fmt.Sprintf("ok-pw-%d@example.com", time.Now().UnixNano())
	okPassword := "valid-pass"
	okResp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    okEmail,
		"name":     "Valid Password",
		"password": okPassword,
		"role":     "viewer",
	}, adminCookies)
	if okResp.Code != http.StatusCreated {
		t.Fatalf("valid password status=%d want 201 body=%s", okResp.Code, okResp.Body.String())
	}

	userCookies := login(t, handler, okEmail, okPassword)
	meResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, userCookies)
	if meResp.Code != http.StatusOK {
		t.Fatalf("login after create status=%d want 200", meResp.Code)
	}
}

func TestSetUserPasswordValidation(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	email := fmt.Sprintf("reset-pw-%d@example.com", time.Now().UnixNano())
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    email,
		"name":     "Reset Target",
		"password": "initial-pass",
		"role":     "viewer",
	}, adminCookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create user status=%d want 201 body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	userID, ok := created["id"].(string)
	if !ok || userID == "" {
		t.Fatalf("expected user id in create response")
	}

	shortResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+userID+"/password", map[string]string{
		"password": "short",
	}, adminCookies)
	if shortResp.Code != http.StatusBadRequest {
		t.Fatalf("short password status=%d want 400 body=%s", shortResp.Code, shortResp.Body.String())
	}

	newPassword := "new-pass-8"
	resetResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+userID+"/password", map[string]string{
		"password": newPassword,
	}, adminCookies)
	if resetResp.Code != http.StatusNoContent {
		t.Fatalf("reset password status=%d want 204 body=%s", resetResp.Code, resetResp.Body.String())
	}

	oldLoginResp := doJSON(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "initial-pass",
	}, nil)
	if oldLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("old password login status=%d want 401", oldLoginResp.Code)
	}

	newCookies := login(t, handler, email, newPassword)
	meResp := doJSON(t, handler, http.MethodGet, "/api/v1/auth/me", nil, newCookies)
	if meResp.Code != http.StatusOK {
		t.Fatalf("new password login status=%d want 200", meResp.Code)
	}
}

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/google/uuid"
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

func TestGetUser(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	email := fmt.Sprintf("get-user-%d@example.com", time.Now().UnixNano())
	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    email,
		"name":     "Get User Test",
		"password": "valid-pass",
		"role":     "viewer",
	}, adminCookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	userID := created["id"].(string)

	getResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/"+userID, nil, adminCookies)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get user status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(getResp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got["id"] != userID {
		t.Fatalf("id=%v want %s", got["id"], userID)
	}
	if got["email"] != email {
		t.Fatalf("email=%v want %s", got["email"], email)
	}
	if got["role"] != "viewer" {
		t.Fatalf("role=%v want viewer", got["role"])
	}
	if _, hasPassword := got["password_hash"]; hasPassword {
		t.Fatalf("response must not include password_hash")
	}
	if _, hasPassword := got["password"]; hasPassword {
		t.Fatalf("response must not include password")
	}

	badIDResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/not-a-uuid", nil, adminCookies)
	if badIDResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid user id status=%d want 400", badIDResp.Code)
	}

	missingID := "00000000-0000-0000-0000-000000000000"
	notFoundResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/"+missingID, nil, adminCookies)
	if notFoundResp.Code != http.StatusNotFound {
		t.Fatalf("missing user status=%d want 404", notFoundResp.Code)
	}

	unauthResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/"+userID, nil, nil)
	if unauthResp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated get user status=%d want 401", unauthResp.Code)
	}

	viewerEmail := fmt.Sprintf("get-user-viewer-%d@example.com", time.Now().UnixNano())
	createViewer := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    viewerEmail,
		"name":     "User Viewer",
		"password": "viewer-password",
		"role":     "viewer",
	}, adminCookies)
	if createViewer.Code != http.StatusCreated {
		t.Fatalf("create viewer status=%d body=%s", createViewer.Code, createViewer.Body.String())
	}
	viewerCookies := login(t, handler, viewerEmail, "viewer-password")

	viewerGetResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/"+userID, nil, viewerCookies)
	if viewerGetResp.Code != http.StatusForbidden {
		t.Fatalf("viewer get user status=%d want 403", viewerGetResp.Code)
	}

	operatorEmail := fmt.Sprintf("get-user-operator-%d@example.com", time.Now().UnixNano())
	createOperator := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    operatorEmail,
		"name":     "User Operator",
		"password": "operator-password",
		"role":     "operator",
	}, adminCookies)
	if createOperator.Code != http.StatusCreated {
		t.Fatalf("create operator status=%d body=%s", createOperator.Code, createOperator.Body.String())
	}
	operatorCookies := login(t, handler, operatorEmail, "operator-password")

	operatorGetResp := doJSON(t, handler, http.MethodGet, "/api/v1/users/"+userID, nil, operatorCookies)
	if operatorGetResp.Code != http.StatusForbidden {
		t.Fatalf("operator get user status=%d want 403", operatorGetResp.Code)
	}
}

func TestLastAdminGuardSession(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)
	admin := testEnvAdmin(t, env)
	isolateSoleEnabledAdmin(t, env.Store, admin.ID)

	operatorDemote := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String(), map[string]any{
		"email": admin.Email,
		"name":  admin.Name,
		"role":  "operator",
	}, adminCookies)
	if operatorDemote.Code != http.StatusBadRequest {
		t.Fatalf("demote last admin to operator status=%d want 400 body=%s", operatorDemote.Code, operatorDemote.Body.String())
	}
	if got := errorMessage(t, operatorDemote); got != "cannot demote the last admin" {
		t.Fatalf("demote last admin error=%q", got)
	}

	viewerDemote := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String(), map[string]any{
		"email": admin.Email,
		"name":  admin.Name,
		"role":  "viewer",
	}, adminCookies)
	if viewerDemote.Code != http.StatusBadRequest {
		t.Fatalf("demote last admin to viewer status=%d want 400 body=%s", viewerDemote.Code, viewerDemote.Body.String())
	}
	if got := errorMessage(t, viewerDemote); got != "cannot demote the last admin" {
		t.Fatalf("demote last admin to viewer error=%q", got)
	}

	disableSelf := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String()+"/disable", nil, adminCookies)
	if disableSelf.Code != http.StatusBadRequest {
		t.Fatalf("disable last admin self status=%d want 400 body=%s", disableSelf.Code, disableSelf.Body.String())
	}
	if got := errorMessage(t, disableSelf); got != "cannot disable your own account" {
		t.Fatalf("disable last admin self error=%q want cannot disable your own account", got)
	}

	updatedName := admin.Name + " Updated"
	keepAdmin := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String(), map[string]any{
		"email": admin.Email,
		"name":  updatedName,
		"role":  "admin",
	}, adminCookies)
	if keepAdmin.Code != http.StatusOK {
		t.Fatalf("email/name update of last admin status=%d want 200 body=%s", keepAdmin.Code, keepAdmin.Body.String())
	}
	var updated map[string]any
	if err := json.Unmarshal(keepAdmin.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode last admin update: %v", err)
	}
	if updated["name"] != updatedName {
		t.Fatalf("name=%v want %s", updated["name"], updatedName)
	}
	if updated["role"] != "admin" {
		t.Fatalf("role=%v want admin", updated["role"])
	}
}

func TestLastAdminGuardAPIToken(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	admin := testEnvAdmin(t, env)
	isolateSoleEnabledAdmin(t, env.Store, admin.ID)
	token := createTestAPIToken(t, env.Store, "last-admin-"+fmt.Sprint(time.Now().UnixNano()), store.RoleAdmin)

	disableResp := doJSONWithToken(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String()+"/disable", nil, token)
	if disableResp.Code != http.StatusBadRequest {
		t.Fatalf("token disable last admin status=%d want 400 body=%s", disableResp.Code, disableResp.Body.String())
	}
	if got := errorMessage(t, disableResp); got != "cannot disable the last admin" {
		t.Fatalf("token disable last admin error=%q", got)
	}

	deleteResp := doJSONWithToken(t, handler, http.MethodDelete, "/api/v1/users/"+admin.ID.String(), nil, token)
	if deleteResp.Code != http.StatusBadRequest {
		t.Fatalf("token delete last admin status=%d want 400 body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if got := errorMessage(t, deleteResp); got != "cannot delete the last admin" {
		t.Fatalf("token delete last admin error=%q", got)
	}

	demoteResp := doJSONWithToken(t, handler, http.MethodPatch, "/api/v1/users/"+admin.ID.String(), map[string]any{
		"email": admin.Email,
		"name":  admin.Name,
		"role":  "operator",
	}, token)
	if demoteResp.Code != http.StatusBadRequest {
		t.Fatalf("token demote last admin status=%d want 400 body=%s", demoteResp.Code, demoteResp.Body.String())
	}
	if got := errorMessage(t, demoteResp); got != "cannot demote the last admin" {
		t.Fatalf("token demote last admin error=%q", got)
	}
}

func TestLastAdminGuardAllowsExtraAdminMutations(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)
	admin := testEnvAdmin(t, env)
	isolateSoleEnabledAdmin(t, env.Store, admin.ID)

	demoteUser := createAPIUser(t, handler, adminCookies, "extra-demote", "admin")
	demoteResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+demoteUser["id"].(string), map[string]any{
		"email": demoteUser["email"],
		"name":  demoteUser["name"],
		"role":  "operator",
	}, adminCookies)
	if demoteResp.Code != http.StatusOK {
		t.Fatalf("demote extra admin status=%d want 200 body=%s", demoteResp.Code, demoteResp.Body.String())
	}

	disableUser := createAPIUser(t, handler, adminCookies, "extra-disable", "admin")
	disableResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+disableUser["id"].(string)+"/disable", nil, adminCookies)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable extra admin status=%d want 200 body=%s", disableResp.Code, disableResp.Body.String())
	}

	deleteUser := createAPIUser(t, handler, adminCookies, "extra-delete", "admin")
	deleteResp := doJSON(t, handler, http.MethodDelete, "/api/v1/users/"+deleteUser["id"].(string), nil, adminCookies)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete extra admin status=%d want 204 body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestLastAdminGuardAllowsNonAdminMutations(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)
	admin := testEnvAdmin(t, env)
	isolateSoleEnabledAdmin(t, env.Store, admin.ID)

	viewer := createAPIUser(t, handler, adminCookies, "last-admin-viewer", "viewer")
	viewerID := viewer["id"].(string)

	updateResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+viewerID, map[string]any{
		"email": viewer["email"],
		"name":  "Viewer Updated",
		"role":  "operator",
	}, adminCookies)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update viewer status=%d want 200 body=%s", updateResp.Code, updateResp.Body.String())
	}

	disableResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+viewerID+"/disable", nil, adminCookies)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable viewer status=%d want 200 body=%s", disableResp.Code, disableResp.Body.String())
	}

	enableResp := doJSON(t, handler, http.MethodPatch, "/api/v1/users/"+viewerID+"/enable", nil, adminCookies)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable viewer status=%d want 200 body=%s", enableResp.Code, enableResp.Body.String())
	}

	deleteResp := doJSON(t, handler, http.MethodDelete, "/api/v1/users/"+viewerID, nil, adminCookies)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete viewer status=%d want 204 body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func testEnvAdmin(t *testing.T, env testEnv) *store.User {
	t.Helper()
	user, err := env.Store.GetUserByEmail(context.Background(), env.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if user == nil {
		t.Fatalf("test env admin %s not found", env.Email)
	}
	return user
}

func isolateSoleEnabledAdmin(t *testing.T, st *store.Store, keepID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	type demotedAdmin struct {
		id   uuid.UUID
		role store.Role
	}
	var demoted []demotedAdmin

	const pageSize = 100
	offset := 0
	for {
		users, total, err := st.ListUsers(ctx, store.ListParams{
			Sort:   "created_at",
			Order:  "asc",
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		for _, u := range users {
			if u.ID == keepID || u.Role != store.RoleAdmin || u.DisabledAt != nil {
				continue
			}
			if _, err := st.UpdateUser(ctx, u.ID, u.Email, u.Name, store.RoleOperator); err != nil {
				t.Fatalf("demote leftover admin %s: %v", u.ID, err)
			}
			demoted = append(demoted, demotedAdmin{id: u.ID, role: store.RoleAdmin})
		}
		offset += len(users)
		if len(users) == 0 || offset >= int(total) {
			break
		}
	}

	t.Cleanup(func() {
		for _, d := range demoted {
			existing, err := st.GetUserByID(ctx, d.id)
			if err != nil || existing == nil {
				continue
			}
			_, _ = st.UpdateUser(ctx, existing.ID, existing.Email, existing.Name, d.role)
		}
	})
}

func createAPIUser(t *testing.T, handler http.Handler, cookies []*http.Cookie, prefix, role string) map[string]any {
	t.Helper()
	email := fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
	resp := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    email,
		"name":     prefix,
		"password": "valid-pass",
		"role":     role,
	}, cookies)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create %s user status=%d body=%s", role, resp.Code, resp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create %s user: %v", role, err)
	}
	return created
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v body=%s", err, rec.Body.String())
	}
	msg, _ := body["error"].(string)
	return msg
}

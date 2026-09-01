package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/store"
)

func TestAPITokenCRUD(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/api-tokens", map[string]any{
		"name": "terraform",
		"role": "operator",
	}, adminCookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create api token status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	rawToken, ok := created["token"].(string)
	if !ok || rawToken == "" {
		t.Fatalf("expected raw token in create response: %#v", created["token"])
	}
	tokenID, ok := created["id"].(string)
	if !ok || tokenID == "" {
		t.Fatalf("expected token id in create response")
	}

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/api-tokens", nil, adminCookies)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list api tokens status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	meResp := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/auth/me", nil, rawToken)
	if meResp.Code != http.StatusOK {
		t.Fatalf("auth/me with bearer status=%d body=%s", meResp.Code, meResp.Body.String())
	}
	var me map[string]any
	if err := json.Unmarshal(meResp.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode auth/me: %v", err)
	}
	if me["auth_method"] != string(auth.AuthMethodAPIToken) {
		t.Fatalf("auth_method=%v want api_token", me["auth_method"])
	}

	revokeResp := doJSON(t, handler, http.MethodPatch, "/api/v1/api-tokens/"+tokenID+"/revoke", nil, adminCookies)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("revoke api token status=%d body=%s", revokeResp.Code, revokeResp.Body.String())
	}

	afterRevoke := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/licenses", nil, rawToken)
	if afterRevoke.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status=%d want 401", afterRevoke.Code)
	}

	deleteResp := doJSON(t, handler, http.MethodDelete, "/api/v1/api-tokens/"+tokenID, nil, adminCookies)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete api token status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestAPITokenManagementRequiresAdminSession(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	viewerEmail := fmt.Sprintf("viewer-token-%d@example.com", time.Now().UnixNano())
	createViewer := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    viewerEmail,
		"name":     "Viewer",
		"password": "viewer-password",
		"role":     "viewer",
	}, adminCookies)
	if createViewer.Code != http.StatusCreated {
		t.Fatalf("create viewer status=%d", createViewer.Code)
	}
	viewerCookies := login(t, handler, viewerEmail, "viewer-password")

	listResp := doJSON(t, handler, http.MethodGet, "/api/v1/api-tokens", nil, viewerCookies)
	if listResp.Code != http.StatusForbidden {
		t.Fatalf("viewer list api tokens status=%d want 403", listResp.Code)
	}

	operatorEmail := fmt.Sprintf("operator-token-%d@example.com", time.Now().UnixNano())
	createOperator := doJSON(t, handler, http.MethodPost, "/api/v1/users", map[string]any{
		"email":    operatorEmail,
		"name":     "Operator",
		"password": "operator-password",
		"role":     "operator",
	}, adminCookies)
	if createOperator.Code != http.StatusCreated {
		t.Fatalf("create operator status=%d", createOperator.Code)
	}
	operatorCookies := login(t, handler, operatorEmail, "operator-password")

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/api-tokens", map[string]any{
		"name": "blocked",
		"role": "viewer",
	}, operatorCookies)
	if createResp.Code != http.StatusForbidden {
		t.Fatalf("operator create api token status=%d want 403", createResp.Code)
	}
}

func TestAPITokenCannotManageTokens(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	createResp := doJSON(t, handler, http.MethodPost, "/api/v1/api-tokens", map[string]any{
		"name": "self-manage-" + fmt.Sprint(time.Now().UnixNano()),
		"role": "admin",
	}, adminCookies)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create api token status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	rawToken := created["token"].(string)

	listResp := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/api-tokens", nil, rawToken)
	if listResp.Code != http.StatusForbidden {
		t.Fatalf("token list api tokens status=%d want 403", listResp.Code)
	}
}

func TestBearerTokenAuth(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	operatorToken := createTestAPIToken(t, env.Store, "operator-"+fmt.Sprint(time.Now().UnixNano()), store.RoleOperator)
	viewerToken := createTestAPIToken(t, env.Store, "viewer-"+fmt.Sprint(time.Now().UnixNano()), store.RoleViewer)

	listResp := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/licenses", nil, operatorToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("operator bearer list licenses status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	productCode := fmt.Sprintf("bearer-product-%d", time.Now().UnixNano())
	createProduct := doJSONWithToken(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Bearer Product",
		"code": productCode,
	}, operatorToken)
	if createProduct.Code != http.StatusCreated {
		t.Fatalf("operator bearer create product status=%d body=%s", createProduct.Code, createProduct.Body.String())
	}

	viewerWrite := doJSONWithToken(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Forbidden",
		"code": "forbidden-" + fmt.Sprint(time.Now().UnixNano()),
	}, viewerToken)
	if viewerWrite.Code != http.StatusForbidden {
		t.Fatalf("viewer bearer create product status=%d want 403", viewerWrite.Code)
	}

	garbageResp := doJSONWithToken(t, handler, http.MethodGet, "/api/v1/licenses", nil, "olsd_not-a-real-token")
	if garbageResp.Code != http.StatusUnauthorized {
		t.Fatalf("garbage bearer status=%d want 401", garbageResp.Code)
	}

	logoutResp := doJSONWithToken(t, handler, http.MethodPost, "/api/v1/auth/logout", nil, operatorToken)
	if logoutResp.Code != http.StatusForbidden {
		t.Fatalf("bearer logout status=%d want 403", logoutResp.Code)
	}

	_ = adminCookies
}

func TestBearerTokenCreateLicenseWithoutCreatedBy(t *testing.T) {
	env := setupTestEnv(t)
	handler := env.Handler
	adminCookies := login(t, handler, env.Email, env.Password)

	productCode := fmt.Sprintf("token-license-%d", time.Now().UnixNano())
	createProduct := doJSON(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Token License Product",
		"code": productCode,
	}, adminCookies)
	if createProduct.Code != http.StatusCreated {
		t.Fatalf("create product status=%d", createProduct.Code)
	}
	var product map[string]any
	if err := json.Unmarshal(createProduct.Body.Bytes(), &product); err != nil {
		t.Fatalf("decode product: %v", err)
	}

	createPolicy := doJSON(t, handler, http.MethodPost, "/api/v1/policies", map[string]any{
		"product_id":       product["id"],
		"name":             "Perpetual",
		"expiration_basis": "on_creation",
	}, adminCookies)
	if createPolicy.Code != http.StatusCreated {
		t.Fatalf("create policy status=%d", createPolicy.Code)
	}
	var policy map[string]any
	if err := json.Unmarshal(createPolicy.Body.Bytes(), &policy); err != nil {
		t.Fatalf("decode policy: %v", err)
	}

	operatorToken := createTestAPIToken(t, env.Store, "license-creator-"+fmt.Sprint(time.Now().UnixNano()), store.RoleOperator)
	createLicense := doJSONWithToken(t, handler, http.MethodPost, "/api/v1/licenses", map[string]any{
		"label":      "token-created",
		"product_id": product["id"],
		"policy_id":  policy["id"],
	}, operatorToken)
	if createLicense.Code != http.StatusCreated {
		t.Fatalf("bearer create license status=%d body=%s", createLicense.Code, createLicense.Body.String())
	}

	var license map[string]any
	if err := json.Unmarshal(createLicense.Body.Bytes(), &license); err != nil {
		t.Fatalf("decode license: %v", err)
	}
	if license["created_by"] != nil {
		t.Fatalf("created_by=%v want null for api token auth", license["created_by"])
	}
}
